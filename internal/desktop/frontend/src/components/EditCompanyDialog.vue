<template>
  <q-dialog v-model="isOpen" :no-backdrop-dismiss="dirty" :persistent="loading">
    <q-card style="min-width: 460px">
      <q-form @submit="submit">
        <q-card-section>
          <div class="text-h6">Editar Empresa</div>
        </q-card-section>

        <q-card-section class="q-pt-none q-gutter-y-md">
          <DraftBanner v-if="restored" @clear="clearDraft" />
          <q-input
            v-model="form.CNPJ"
            label="CNPJ"
            outlined
            dense
            readonly
            hint="O CNPJ não pode ser alterado."
          />
          <q-input
            v-model="form.Name"
            label="Nome / Razão Social"
            outlined
            dense
            autofocus
            :error="submitted && !!errors.Name"
            :error-message="errors.Name"
          />

          <q-select
            v-model="form.Environment"
            :options="environmentOptions"
            emit-value
            map-options
            label="Ambiente"
            outlined
            dense
          />

          <q-select
            v-model="form.UF"
            :options="ufOptions"
            label="UF (opcional)"
            hint="Necessária para buscar NF-e na SEFAZ."
            clearable
            outlined
            dense
          />

          <q-separator />

          <q-select
            v-model="form.SyncStartPolicy"
            :options="syncPolicyOptions"
            label="Política inicial de histórico"
            emit-value
            map-options
            outlined
            dense
            :disable="!canEditSyncPolicy"
          />

          <q-input
            v-if="form.SyncStartPolicy === 'since_date'"
            v-model="form.SyncStartDate"
            label="Importar a partir de"
            type="date"
            outlined
            dense
            :readonly="!canEditSyncPolicy"
            :error="submitted && !!errors.SyncStartDate"
            :error-message="errors.SyncStartDate"
          />

          <q-banner
            v-if="!canEditSyncPolicy"
            dense
            rounded
            :class="mutedBannerClass($q.dark.isActive)"
          >
            A data inicial fica travada depois que a NFS-e começou a sincronizar. Para trocá-la, use
            Redefinir NFS-e na página NFS-e: a próxima sincronização baixa de novo desde o início, a
            partir da nova data.
          </q-banner>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn v-close-popup flat label="Cancelar" color="primary" :disable="loading" />
          <q-btn flat type="submit" label="Salvar" color="primary" :loading="loading" />
        </q-card-actions>
      </q-form>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import DraftBanner from '@/components/DraftBanner.vue'
import { useFormDraft } from '@/composables/useFormDraft'
import { useCompanies } from '@/composables/useCompanies'
import { errorMessage } from '@/platform/wails/errors'
import type { CompanySummary, SyncStartPolicy } from '@/types/desktop'
import { environmentOptions } from '@/utils/nfseDisplay'
import { UF_SIGLAS } from '@/utils/uf'
import { mutedBannerClass } from '@/utils/bannerClass'

const props = defineProps<{
  modelValue: boolean
  companyData: CompanySummary | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'updated'): void
}>()
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const { updateCompany } = useCompanies()
const loading = ref(false)
// submitted turns the inline errors on after the first save attempt.
const submitted = ref(false)
const ufOptions = [...UF_SIGLAS]

// companySeed is the form as the company is stored.
function companySeed() {
  const company = props.companyData
  return {
    CNPJ: company?.CNPJ || '',
    Name: company?.Name || '',
    Environment: company?.Environment || 'producao',
    // q-select sets null when cleared.
    UF: (company?.UF || '') as string | null,
    SyncStartPolicy: (company?.SyncStartPolicy || 'from_now') as SyncStartPolicy,
    SyncStartDate: formatInputDate(company?.SyncStartDate),
  }
}

const { form, dirty, restored, open, keep, clear, done } = useFormDraft(
  () => `company:edit:${props.companyData?.CNPJ ?? ''}`,
  companySeed,
  { dropWhenSeedChanges: true }
)

const errors = computed(() => {
  const found: Partial<Record<'Name' | 'SyncStartDate', string>> = {}
  if (!form.value.Name) found.Name = 'Informe o nome da empresa.'
  if (form.value.SyncStartPolicy === 'since_date' && !form.value.SyncStartDate) {
    found.SyncStartDate = 'Informe a data inicial de importação.'
  }
  return found
})

const syncPolicyOptions = [
  { label: 'Todo o histórico disponível', value: 'all' },
  { label: 'A partir de uma data específica', value: 'since_date' },
  { label: 'A partir de hoje', value: 'from_now' },
]

// canEditSyncPolicy follows the NFS-e cursor, like the backend: Redefinir
// NFS-e deletes it and unlocks the start date, while LastSyncAt stays.
const canEditSyncPolicy = computed(() => !props.companyData?.SyncStartPolicyLocked)

watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
    if (val) {
      submitted.value = false
      open()
    }
  }
)

watch(isOpen, (val) => {
  if (!val) keep()
  emit('update:modelValue', val)
})

function clearDraft() {
  submitted.value = false
  clear()
}

async function submit() {
  if (loading.value) return
  submitted.value = true
  if (Object.keys(errors.value).length > 0) return

  loading.value = true
  try {
    await updateCompany({
      CNPJ: form.value.CNPJ,
      Name: form.value.Name,
      Environment: form.value.Environment,
      UF: form.value.UF ?? '',
      SyncStartPolicy: form.value.SyncStartPolicy,
      SyncStartDate: syncStartDateForSubmit(),
    })
    $q.notify({ type: 'positive', message: 'Empresa atualizada com sucesso!' })
    emit('updated')
    done()
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atualizar empresa: ' + errorMessage(err) })
  } finally {
    loading.value = false
  }
}

function syncStartDateForSubmit() {
  if (form.value.SyncStartPolicy === 'all') return ''
  if (
    form.value.SyncStartPolicy === 'from_now' &&
    props.companyData?.SyncStartPolicy !== 'from_now'
  ) {
    return ''
  }
  return form.value.SyncStartDate
}

function formatInputDate(value: CompanySummary['SyncStartDate']) {
  if (!value) return ''
  if (value instanceof Date) return value.toISOString().slice(0, 10)
  return String(value).slice(0, 10)
}
</script>
