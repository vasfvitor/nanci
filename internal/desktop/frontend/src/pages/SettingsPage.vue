<template>
  <q-page padding class="q-gutter-y-md">
    <div class="row items-center">
      <div class="text-h5 text-weight-bold">Configurações e Suporte</div>
      <q-space />
    </div>

    <!-- Section 1: Preferences -->
    <q-card flat bordered class="q-pa-md shadow-1">
      <div class="text-subtitle1 text-weight-bold q-mb-md text-primary">Preferências de Interface</div>
      <q-list>
        <q-item tag="label" class="q-px-none">
          <q-item-section>
            <q-item-label class="text-weight-medium">Modo Escuro (Dark Mode)</q-item-label>
            <q-item-label caption>Ajuste o tema de cores do aplicativo</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-select
              v-model="darkMode"
              :options="darkModeOptions"
              emit-value
              map-options
              dense
              outlined
              style="min-width: 200px"
              @update:model-value="updateDarkMode"
            />
          </q-item-section>
        </q-item>
        <q-separator spaced />
        <q-item tag="label" v-ripple class="q-px-none">
          <q-item-section>
            <q-item-label class="text-weight-medium">Modo Debug</q-item-label>
            <q-item-label caption>Habilita logs detalhados e ações de desenvolvedor (ex: Resetar NSU)</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-toggle v-model="debugToggle" color="primary" />
          </q-item-section>
        </q-item>
      </q-list>
    </q-card>

    <!-- Section 2: Connection Test -->
    <q-card flat bordered class="q-pa-md shadow-1">
      <div class="text-subtitle1 text-weight-bold q-mb-md text-primary">Ferramenta de Diagnóstico de Conexão</div>
      <div class="q-mb-md text-body2 text-grey-7">
        Selecione uma empresa para testar a comunicação criptografada (mTLS) e a resposta da API do governo.
      </div>
      <div class="row q-col-gutter-sm items-center">
        <q-select
          v-model="selectedCompany"
          class="col-12 col-md-5"
          :options="companyOptions"
          label="Empresa"
          emit-value
          map-options
          outlined
          dense
          options-dense
        />
        <div class="col-12 col-md-3">
          <q-btn
            color="primary"
            icon="network_check"
            label="Testar Conexão"
            :disable="!selectedCompany || testing"
            :loading="testing"
            class="full-width"
            @click="runConnectionTest"
          />
        </div>
      </div>

      <!-- Connection Test Results -->
      <div v-if="testResult" :class="['q-mt-md q-pa-md rounded-borders text-body2 border-dashed', $q.dark.isActive ? 'bg-grey-9 text-grey-3' : 'bg-grey-1 text-grey-8']">
        <div class="text-subtitle2 text-weight-bold q-mb-sm">Resultado do Diagnóstico:</div>
        <div class="row q-col-gutter-sm">
          <div class="col-6">
            <strong>Certificado Carregado:</strong>
            <q-icon :name="testResult.certLoaded ? 'check_circle' : 'cancel'" :color="testResult.certLoaded ? 'positive' : 'negative'" class="q-ml-xs" size="xs" />
          </div>
          <div v-if="testResult.certLoaded" class="col-6">
            <strong>Expiração:</strong> <span class="text-mono">{{ testResult.certExpiration }}</span>
          </div>
          <div v-if="testResult.certLoaded" class="col-12">
            <strong>Assunto:</strong> <span class="text-caption text-grey-6">{{ testResult.certSubject }}</span>
          </div>
          <div class="col-6">
            <strong>mTLS Aceito:</strong>
            <q-icon :name="testResult.mtlsAccepted ? 'check_circle' : 'cancel'" :color="testResult.mtlsAccepted ? 'positive' : 'negative'" class="q-ml-xs" size="xs" />
          </div>
          <div class="col-6">
            <strong>Endpoint ADN Alcançado:</strong>
            <q-icon :name="testResult.endpointReached ? 'check_circle' : 'cancel'" :color="testResult.endpointReached ? 'positive' : 'negative'" class="q-ml-xs" size="xs" />
          </div>
          <div class="col-12">
            <strong>Resposta ADN:</strong> <span class="text-mono text-weight-medium">{{ testResult.responseCode }}</span>
            <div class="text-caption text-grey-6 q-mt-xs">{{ testResult.responseDetail }}</div>
          </div>
          <q-separator class="col-12 q-my-xs" />
          <div class="col-12">
            <strong>Conclusão:</strong>
            <div class="text-subtitle2 text-weight-bold q-mt-xs" :class="testResult.endpointReached ? 'text-positive' : 'text-negative'">
              {{ testResult.statusExplanation }}
            </div>
          </div>
        </div>
      </div>
    </q-card>

    <!-- Section 3: About & Diagnostics -->
    <q-card flat bordered class="q-pa-md shadow-1">
      <div class="text-subtitle1 text-weight-bold q-mb-md text-primary">Sobre o Nanci</div>
      <div class="row q-col-gutter-md">
        <!-- System Metadata -->
        <div class="col-12 col-md-6 text-body2" :class="$q.dark.isActive ? 'text-grey-3' : 'text-grey-8'">
          <div class="row q-col-gutter-xs">
            <div class="col-4 text-weight-bold">Versão:</div>
            <div class="col-8 text-mono">{{ buildInfo.version }}</div>
            <div class="col-4 text-weight-bold">Commit:</div>
            <div class="col-8 text-mono">{{ buildInfo.commit }}</div>
            <div class="col-4 text-weight-bold">Build:</div>
            <div class="col-8 text-mono">{{ buildInfo.date }}</div>
            <div class="col-4 text-weight-bold">Pasta de Dados:</div>
            <div class="col-8 text-mono text-caption ellipsis" :title="dataDir">{{ dataDir }}</div>
          </div>
        </div>

        <!-- Diagnostic Actions -->
        <div class="col-12 col-md-6 column justify-center q-gutter-y-sm">
          <div class="row q-col-gutter-sm">
            <div class="col-6">
              <q-btn
                flat
                bordered
                dense
                icon="folder"
                label="Abrir Pasta de Dados"
                class="full-width text-body2"
                @click="openDataDir"
              />
            </div>
            <div class="col-6">
              <q-btn
                flat
                bordered
                dense
                icon="file_open"
                label="Abrir Pasta de Logs"
                class="full-width text-body2"
                @click="openLogsDir"
              />
            </div>
            <div class="col-12">
              <q-btn
                color="secondary"
                icon="archive"
                label="Exportar Pacote de Diagnóstico (Logs)"
                class="full-width"
                @click="exportDiagnosticLogs"
              />
            </div>
          </div>
        </div>
      </div>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import { desktopRuntime } from '@/platform/wails/runtime'
import { desktopClient } from '@/platform/wails/client'
import { useDiagnosticsStore } from '@/stores/diagnostics'
import { useConsoleStore } from '@/stores/console'
import type { BuildInfo } from '@/types/desktop'

const $q = useQuasar()
const diagnosticsStore = useDiagnosticsStore()
const { testing, testResult } = storeToRefs(diagnosticsStore)

const consoleStore = useConsoleStore()

const debugToggle = computed({
  get: () => consoleStore.debugEnabled,
  set: (val: boolean) => {
    consoleStore.setLogFilter(val ? 'debug' : 'info')
  }
})

const darkMode = ref<'auto' | boolean>('auto')

const buildInfo = ref<BuildInfo>({ version: '...', commit: '...', date: '...' })
const dataDir = ref('')
const selectedCompany = ref('')
const companyOptions = ref<{ label: string; value: string }[]>([])

const darkModeOptions = [
  { label: 'Automático (Sistema)', value: 'auto' },
  { label: 'Sempre Claro', value: false },
  { label: 'Sempre Escuro', value: true }
]

async function loadCompanies() {
  try {
    const companies = await desktopClient.listCompanies()
    companyOptions.value = companies.map((company) => ({
      label: `${company.Name} (${company.CNPJ})`,
      value: company.CNPJ,
    }))
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar empresas: ' + String(err) })
  }
}

onMounted(async () => {
  const saved = localStorage.getItem('darkMode')
  if (saved === 'true') {
    darkMode.value = true
  } else if (saved === 'false') {
    darkMode.value = false
  } else {
    darkMode.value = 'auto'
  }

  try {
    buildInfo.value = await desktopClient.getBuildInfo()
    dataDir.value = await desktopClient.getDataDirectory()
    await loadCompanies()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar dados de sistema: ' + String(err) })
  }
})

function updateDarkMode(val: 'auto' | boolean) {
  localStorage.setItem('darkMode', String(val))
  $q.dark.set(val)
  if (val === true) {
    desktopRuntime.setDarkTheme()
  } else if (val === false) {
    desktopRuntime.setLightTheme()
  } else {
    desktopRuntime.setSystemDefaultTheme()
  }
}

async function openDataDir() {
  try {
    await desktopClient.openDataDirectory()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao abrir pasta de dados: ' + String(err) })
  }
}

async function openLogsDir() {
  try {
    await desktopClient.openLogsDirectory()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao abrir pasta de logs: ' + String(err) })
  }
}

async function exportDiagnosticLogs() {
  try {
    const result = await desktopClient.exportLogs()
    if (result) {
      $q.notify({ type: 'positive', message: 'Logs exportados com sucesso para ' + result })
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao exportar logs: ' + String(err) })
  }
}

async function runConnectionTest() {
  if (!selectedCompany.value) return
  testing.value = true
  diagnosticsStore.clearResult()
  try {
    testResult.value = await desktopClient.testConnection(selectedCompany.value)
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Falha ao executar teste de conexão: ' + String(err) })
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.border-dashed {
  border: 1px dashed var(--q-separator-color, rgba(0, 0, 0, 0.15));
}
</style>
