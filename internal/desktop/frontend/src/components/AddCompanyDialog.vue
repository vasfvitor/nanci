<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 560px">
      <q-card-section>
        <div class="text-h6">Adicionar Empresa</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input
          v-model="form.CNPJ"
          label="CNPJ"
          hint="Use um CNPJ numérico com DV válido. CNPJ alfanumérico ainda não está liberado nesta versão."
          outlined
          dense
        />
        <q-input v-model="form.Name" label="Nome / Razão Social" outlined dense />
        <q-select
          v-model="form.Environment"
          :options="['producao', 'producao_restrita']"
          label="Ambiente da Empresa"
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
          v-model="syncStartChoice"
          :options="syncStartOptions"
          color="primary"
        />

        <q-input
          v-if="syncStartChoice === 'custom_date'"
          v-model="customSyncStartDate"
          label="Importar a partir de"
          type="date"
          outlined
          dense
        />

        <q-option-group
          v-model="credentialMode"
          inline
          :options="[
            { label: 'Usar credencial existente', value: 'existing' },
            { label: 'Criar nova credencial', value: 'new' },
          ]"
        />

        <q-select
          v-if="credentialMode === 'existing'"
          v-model="form.CredentialID"
          :options="credentialOptions"
          label="Credencial"
          emit-value
          map-options
          outlined
          dense
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
            />
            <q-btn icon="folder" color="primary" @click="selectCert" />
          </div>
        </template>
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
import { date, useQuasar } from 'quasar'
import { desktopClient } from '@/platform/wails/client'
import type { SyncStartPolicy } from '@/types/desktop'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'added'): void
}>()
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)
const credentialMode = ref<'existing' | 'new'>('existing')
const credentialOptions = ref<{ label: string; value: string }[]>([])
const syncStartChoice = ref<'from_now' | 'last_12_months' | 'last_5_years' | 'custom_date' | 'all'>('from_now')
const customSyncStartDate = ref('')

const form = ref({
  CNPJ: '',
  Name: '',
  CredentialID: '',
  CredentialLabel: '',
  CertPath: '',
  Environment: 'producao_restrita',
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
      resetForm()
      await loadCredentials()
    }
  }
)

watch(isOpen, (val) => {
  emit('update:modelValue', val)
})

async function loadCredentials() {
  try {
    const credentials = await desktopClient.listCredentials()
    credentialOptions.value = credentials.map((credential) => ({
      label: `${credential.Label}`,
      value: credential.ID,
    }))
    credentialMode.value = credentialOptions.value.length > 0 ? 'existing' : 'new'
    if (credentialOptions.value.length > 0 && !form.value.CredentialID) {
      const firstOpt = credentialOptions.value[0]
      if (firstOpt) {
        form.value.CredentialID = firstOpt.value
      }
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar credenciais: ' + String(err) })
  }
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
    $q.notify({ type: 'negative', message: 'Erro ao selecionar certificado: ' + String(err) })
  }
}

async function submit() {
  if (!form.value.CNPJ || !form.value.Name) {
    $q.notify({ type: 'warning', message: 'Preencha CNPJ e nome da empresa.' })
    return
  }
  if (credentialMode.value === 'existing' && !form.value.CredentialID) {
    $q.notify({ type: 'warning', message: 'Selecione uma credencial.' })
    return
  }
  if (credentialMode.value === 'new' && !form.value.CertPath) {
    $q.notify({ type: 'warning', message: 'Selecione um certificado.' })
    return
  }
  const syncStart = resolveSyncStartInput()
  if (!syncStart) return

  loading.value = true
  try {
    await desktopClient.addCompany({
      CNPJ: form.value.CNPJ,
      Name: form.value.Name,
      CredentialID: credentialMode.value === 'existing' ? form.value.CredentialID : '',
      CredentialLabel: credentialMode.value === 'new' ? form.value.CredentialLabel : '',
      CertPath: credentialMode.value === 'new' ? form.value.CertPath : '',
      Environment: form.value.Environment,
      SyncStartPolicy: syncStart.policy,
      SyncStartDate: syncStart.date,
    })
    $q.notify({ type: 'positive', message: 'Empresa adicionada com sucesso!' })
    emit('added')
    isOpen.value = false
    resetForm()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao adicionar empresa: ' + String(err) })
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = {
    CNPJ: '',
    Name: '',
    CredentialID: credentialOptions.value[0]?.value || '',
    CredentialLabel: '',
    CertPath: '',
    Environment: 'producao_restrita',
  }
  syncStartChoice.value = 'from_now'
  customSyncStartDate.value = ''
}

function resolveSyncStartInput(): { policy: SyncStartPolicy; date: string } | null {
  const today = new Date()
  switch (syncStartChoice.value) {
    case 'from_now':
      return { policy: 'from_now', date: '' }
    case 'last_12_months':
      return { policy: 'since_date', date: addYears(today, -1) }
    case 'last_5_years':
      return { policy: 'since_date', date: addYears(today, -5) }
    case 'custom_date':
      if (!customSyncStartDate.value) {
        $q.notify({ type: 'warning', message: 'Informe a data inicial de importação.' })
        return null
      }
      return { policy: 'since_date', date: customSyncStartDate.value }
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
