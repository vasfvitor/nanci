<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 460px">
      <q-card-section>
        <div class="text-h6">Editar Empresa</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input
          v-model="form.CNPJ"
          label="CNPJ"
          outlined
          dense
          readonly
          hint="O CNPJ não pode ser alterado."
        />
        <q-input v-model="form.Name" label="Nome / Razão Social" outlined dense />

        <q-select
          v-model="form.Environment"
          :options="['producao', 'producao_restrita']"
          label="Ambiente"
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
        />

        <q-banner
          v-if="!canEditSyncPolicy"
          dense
          rounded
          :class="$q.dark.isActive ? 'bg-grey-9 text-grey-3' : 'bg-grey-2 text-grey-8'"
        >
          A política inicial não pode ser alterada depois que a sincronização começou. Para
          refazer o bootstrap histórico, use uma ação avançada de reset/backfill.
        </q-banner>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn v-close-popup flat label="Cancelar" color="primary" />
        <q-btn flat label="Salvar" color="primary" :loading="loading" @click="submit" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary, SyncStartPolicy } from '@/types/desktop'

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
const loading = ref(false)

const form = ref({
  CNPJ: '',
  Name: '',
  Environment: 'producao',
  SyncStartPolicy: 'from_now' as SyncStartPolicy,
  SyncStartDate: '',
})

const syncPolicyOptions = [
  { label: 'Todo o histórico disponível', value: 'all' },
  { label: 'A partir de uma data específica', value: 'since_date' },
  { label: 'A partir de hoje', value: 'from_now' },
]

const canEditSyncPolicy = computed(() => {
  const company = props.companyData
  if (!company) return true
  return !company.LastSyncAt && !company.InitialSyncDoneAt
})

watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
    if (val && props.companyData) {
      form.value.CNPJ = props.companyData.CNPJ || ''
      form.value.Name = props.companyData.Name || ''
      form.value.Environment = props.companyData.Environment || 'producao'
      form.value.SyncStartPolicy = props.companyData.SyncStartPolicy || 'from_now'
      form.value.SyncStartDate = formatInputDate(props.companyData.SyncStartDate)
    }
  }
)

watch(isOpen, (val) => {
  emit('update:modelValue', val)
})

async function submit() {
  if (!form.value.Name) {
    $q.notify({ type: 'warning', message: 'Preencha o nome da empresa.' })
    return
  }
  if (form.value.SyncStartPolicy === 'since_date' && !form.value.SyncStartDate) {
    $q.notify({ type: 'warning', message: 'Informe a data inicial de importação.' })
    return
  }

  loading.value = true
  try {
    await desktopClient.updateCompany({
      CNPJ: form.value.CNPJ,
      Name: form.value.Name,
      Environment: form.value.Environment,
      SyncStartPolicy: form.value.SyncStartPolicy,
      SyncStartDate: syncStartDateForSubmit(),
    })
    $q.notify({ type: 'positive', message: 'Empresa atualizada com sucesso!' })
    emit('updated')
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atualizar empresa: ' + String(err) })
  } finally {
    loading.value = false
  }
}

function syncStartDateForSubmit() {
  if (form.value.SyncStartPolicy === 'all') return ''
  if (form.value.SyncStartPolicy === 'from_now' && props.companyData?.SyncStartPolicy !== 'from_now') {
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
