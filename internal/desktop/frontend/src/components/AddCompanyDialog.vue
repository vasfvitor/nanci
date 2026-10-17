<template>
  <q-dialog v-model="isOpen" :no-backdrop-dismiss="dirty" :persistent="loading">
    <q-card style="min-width: 560px">
      <q-form @submit="submit">
        <q-card-section>
          <div class="text-h6">Adicionar Empresa</div>
        </q-card-section>

        <q-card-section class="q-pt-none q-gutter-y-md">
          <DraftBanner v-if="restored" @clear="clearDraft" />
          <q-input
            :model-value="form.CNPJ"
            label="CNPJ"
            hint="Informe o CNPJ com DV válido, numérico ou alfanumérico."
            mask="XX.XXX.XXX/XXXX-##"
            unmasked-value
            outlined
            dense
            autofocus
            :error="!!cnpjMessage"
            :error-message="cnpjMessage"
            @update:model-value="(value) => (form.CNPJ = normalizeCNPJInput(value))"
          />
          <q-input
            v-model="form.Name"
            label="Nome / Razão Social"
            outlined
            dense
            :error="submitted && !!errors.Name"
            :error-message="errors.Name"
          />
          <q-select
            v-model="form.Environment"
            :options="environmentOptions"
            emit-value
            map-options
            label="Ambiente da Empresa"
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

          <div>
            <div class="text-subtitle2">O que deseja importar no primeiro sync?</div>
            <div class="text-caption text-app-muted">
              O primeiro sync pode analisar documentos antigos para alcançar a posição atual do ADN,
              mas só importará o histórico escolhido.
            </div>
          </div>

          <q-option-group
            v-model="form.syncStartChoice"
            :options="syncStartOptions"
            color="primary"
          />

          <q-input
            v-if="form.syncStartChoice === 'custom_date'"
            v-model="form.customSyncStartDate"
            label="Importar a partir de"
            type="date"
            outlined
            dense
            :error="submitted && !!errors.customSyncStartDate"
            :error-message="errors.customSyncStartDate"
          />

          <q-option-group
            v-model="form.credentialMode"
            inline
            :options="[
              { label: 'Usar credencial existente', value: 'existing' },
              { label: 'Criar nova credencial', value: 'new' },
            ]"
          />

          <q-select
            v-if="form.credentialMode === 'existing'"
            v-model="form.CredentialID"
            :options="credentialOptions"
            label="Credencial"
            emit-value
            map-options
            outlined
            dense
            :error="submitted && !!errors.CredentialID"
            :error-message="errors.CredentialID"
          />

          <template v-else>
            <q-input v-model="form.CredentialLabel" label="Rótulo da Credencial" outlined dense />
            <div class="row items-center q-gutter-x-sm">
              <q-input
                v-model="form.CertPath"
                label="Caminho do Certificado (.pfx/.p12)"
                outlined
                dense
                readonly
                class="col"
                :error="submitted && !!errors.CertPath"
                :error-message="errors.CertPath"
              />
              <q-btn icon="folder" color="primary" @click="selectCert" />
            </div>
          </template>
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
import { date, useQuasar } from 'quasar'
import DraftBanner from '@/components/DraftBanner.vue'
import { useFormDraft } from '@/composables/useFormDraft'
import { desktopClient, errorMessage } from '@/platform/wails/client'
import type { SyncStartPolicy } from '@/types/desktop'
import { cnpjError, normalizeCNPJInput } from '@/utils/cnpj'
import { environmentOptions } from '@/utils/nfseDisplay'
import { UF_SIGLAS } from '@/utils/uf'

type SyncStartChoice = 'from_now' | 'last_12_months' | 'last_5_years' | 'custom_date' | 'all'
type Field = 'CNPJ' | 'Name' | 'CredentialID' | 'CertPath' | 'customSyncStartDate'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  added: [cnpj: string]
}>()
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)
// submitted turns the inline errors on after the first save attempt.
const submitted = ref(false)
const credentialOptions = ref<{ label: string; value: string }[]>([])
const ufOptions = [...UF_SIGLAS]

// The form starts at the first loaded credential, or at a new one when there
// is none.
function emptyForm() {
  const firstCredential = credentialOptions.value[0]?.value ?? ''
  return {
    CNPJ: '',
    Name: '',
    CredentialID: firstCredential,
    CredentialLabel: '',
    CertPath: '',
    Environment: 'producao',
    // q-select sets null when cleared.
    UF: '' as string | null,
    syncStartChoice: 'from_now' as SyncStartChoice,
    customSyncStartDate: '',
    credentialMode: (firstCredential ? 'existing' : 'new') as 'existing' | 'new',
  }
}

const { form, dirty, restored, open, keep, clear, done } = useFormDraft('company:add', emptyForm)

const errors = computed(() => {
  const found: Partial<Record<Field, string>> = {}
  const cnpj = cnpjError(form.value.CNPJ)
  if (cnpj) found.CNPJ = cnpj
  if (!form.value.Name) found.Name = 'Informe o nome da empresa.'
  if (form.value.credentialMode === 'existing' && !form.value.CredentialID) {
    found.CredentialID = 'Selecione uma credencial.'
  }
  if (form.value.credentialMode === 'new' && !form.value.CertPath) {
    found.CertPath = 'Selecione um certificado.'
  }
  if (form.value.syncStartChoice === 'custom_date' && !form.value.customSyncStartDate) {
    found.customSyncStartDate = 'Informe a data inicial de importação.'
  }
  return found
})

// cnpjMessage judges a complete CNPJ as it is typed; an empty or short one
// waits for a save attempt.
const cnpjMessage = computed(() => {
  if (!submitted.value && form.value.CNPJ.length < 14) return ''
  return errors.value.CNPJ ?? ''
})

const syncStartOptions = computed(() => [
  { label: 'Apenas documentos a partir de hoje', value: 'from_now' },
  { label: 'Últimos 12 meses', value: 'last_12_months' },
  { label: 'Últimos 5 anos', value: 'last_5_years' },
  { label: 'A partir de uma data específica', value: 'custom_date' },
  { label: 'Todo o histórico disponível', value: 'all' },
])

watch(
  () => props.modelValue,
  async (val) => {
    isOpen.value = val
    if (val) {
      await loadCredentials()
      submitted.value = false
      open()
      keepCredentialInList()
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

async function loadCredentials() {
  try {
    const credentials = await desktopClient.listCredentials()
    credentialOptions.value = credentials.map((credential) => ({
      label: `${credential.Label}`,
      value: credential.ID,
    }))
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar credenciais: ' + errorMessage(err) })
  }
}

// keepCredentialInList points a restored draft whose credential was removed
// since at the first credential, or at a new one when there is none left.
function keepCredentialInList() {
  if (credentialOptions.value.some((option) => option.value === form.value.CredentialID)) return
  const firstCredential = credentialOptions.value[0]?.value ?? ''
  form.value.CredentialID = firstCredential
  if (!firstCredential) form.value.credentialMode = 'new'
}

async function selectCert() {
  try {
    const path = await desktopClient.selectCertificate()
    if (path) {
      form.value.CertPath = path
      if (!form.value.CredentialLabel) {
        form.value.CredentialLabel = path.split(/[\\/]/).pop() || path
      }
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao selecionar certificado: ' + errorMessage(err) })
  }
}

async function submit() {
  if (loading.value) return
  submitted.value = true
  if (Object.keys(errors.value).length > 0) return
  const syncStart = resolveSyncStartInput()

  loading.value = true
  try {
    const cnpj = form.value.CNPJ
    const existing = form.value.credentialMode === 'existing'
    await desktopClient.addCompany({
      CNPJ: cnpj,
      Name: form.value.Name,
      CredentialID: existing ? form.value.CredentialID : '',
      CredentialLabel: existing ? '' : form.value.CredentialLabel,
      CertPath: existing ? '' : form.value.CertPath,
      Environment: form.value.Environment,
      UF: form.value.UF ?? '',
      SyncStartPolicy: syncStart.policy,
      SyncStartDate: syncStart.date,
    })
    $q.notify({ type: 'positive', message: 'Empresa adicionada com sucesso!' })
    emit('added', cnpj)
    done()
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao adicionar empresa: ' + errorMessage(err) })
  } finally {
    loading.value = false
  }
}

function resolveSyncStartInput(): { policy: SyncStartPolicy; date: string } {
  const today = new Date()
  switch (form.value.syncStartChoice) {
    case 'from_now':
      return { policy: 'from_now', date: '' }
    case 'last_12_months':
      return { policy: 'since_date', date: addYears(today, -1) }
    case 'last_5_years':
      return { policy: 'since_date', date: addYears(today, -5) }
    case 'custom_date':
      return { policy: 'since_date', date: form.value.customSyncStartDate }
    case 'all':
      return { policy: 'all', date: '' }
  }
}

function addYears(from: Date, years: number) {
  const copy = new Date(from)
  copy.setFullYear(copy.getFullYear() + years)
  return date.formatDate(copy, 'YYYY-MM-DD')
}
</script>
