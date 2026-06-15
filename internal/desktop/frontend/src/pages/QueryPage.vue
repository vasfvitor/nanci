<template>
  <q-page padding class="column">
    <div class="row items-center q-mb-md">
      <div class="text-h5 text-weight-bold">Consulta Direta (API ADN)</div>
      <q-space />
    </div>

    <q-card flat bordered class="q-pa-md q-mb-md">
      <q-form @submit="runQuery" class="q-gutter-md">
        <div class="row q-col-gutter-md">
          <div class="col-12 col-md-6">
            <q-select
              v-model="appStore.queryForm.cnpj"
              :options="companyOptions"
              use-input
              input-debounce="0"
              @new-value="createValue"
              @filter="filterFn"
              emit-value
              map-options
              label="Empresa / CNPJ (Para autenticação)"
              outlined
              dense
              :rules="[val => !!val || 'CNPJ é obrigatório']"
            />
          </div>
          <div class="col-12 col-md-6">
            <q-input
              v-model="appStore.queryForm.chave"
              label="Chave de Acesso (50 posições)"
              outlined
              dense
              :rules="[val => !!val || 'Chave é obrigatória', val => val.length === 50 || 'Chave deve ter 50 números']"
            />
          </div>
        </div>
        <div class="row q-gutter-sm">
          <q-btn type="submit" color="primary" icon="search" label="Consultar NFSe" :loading="loading" @click="appStore.queryType = 'nfse'" />
          <q-btn type="submit" color="secondary" icon="event" label="Consultar Eventos" :loading="loading" @click="appStore.queryType = 'events'" />
        </div>
      </q-form>
    </q-card>

    <q-card flat bordered class="col column q-mt-md" v-if="appStore.queryResult">
      <q-toolbar class="dense bg-primary text-white">
        <q-toolbar-title class="text-subtitle2">Resultado da Consulta</q-toolbar-title>
        <q-btn flat round dense icon="content_copy" @click="copyResult">
          <q-tooltip>Copiar JSON</q-tooltip>
        </q-btn>
      </q-toolbar>
      <div class="q-pa-md">
        <div class="text-body2" style="font-family: 'Fira Code', monospace; font-size: 13px; white-space: pre-wrap; word-break: break-all; word-wrap: break-word;" v-html="highlightedResult">
        </div>
      </div>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useAppStore } from '../stores/app'
import { ListCompanies } from '../../wailsjs/go/main/App'

const $q = useQuasar()
const appStore = useAppStore()

const loading = ref(false)

const allOptions = ref<{label: string, value: string}[]>([])
const companyOptions = ref<{label: string, value: string}[]>([])

onMounted(async () => {
  try {
    const list = await ListCompanies() || []
    const opts = list.map(c => ({
      label: `${c.Name} (${c.CNPJ})`,
      value: c.CNPJ
    }))
    allOptions.value = opts
    companyOptions.value = opts
  } catch (e) {
    console.error("Erro ao carregar empresas para consulta", e)
  }
})

function createValue(val: string, done: (item: any, mode: 'add' | 'add-unique' | 'toggle') => void) {
  if (val.length > 0) {
    done(val, 'add-unique')
  }
}

function filterFn(val: string, update: (callback: () => void) => void) {
  update(() => {
    const needle = val.toLowerCase()
    companyOptions.value = allOptions.value.filter(
      v => v.label.toLowerCase().indexOf(needle) > -1 || v.value.indexOf(needle) > -1
    )
  })
}

const highlightedResult = computed(() => {
  if (!appStore.queryResult) return ''
  let jsonStr = appStore.queryResult
  jsonStr = jsonStr.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return jsonStr.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, (match) => {
    let cls = 'json-string'
    if (/^"/.test(match)) {
      if (/:$/.test(match)) {
        cls = 'json-key'
      }
    } else if (/true|false/.test(match)) {
      cls = 'json-boolean'
    } else if (/null/.test(match)) {
      cls = 'json-null'
    } else {
      cls = 'json-number'
    }
    return `<span class="${cls}">${match}</span>`
  })
})

async function runQuery() {
  if (!appStore.queryForm.cnpj || appStore.queryForm.chave.length !== 50) return

  loading.value = true
  appStore.queryResult = ''

  try {
    // We use ts-ignore dynamically calling the Wails backend since bindings might not be generated yet
    // @ts-ignore
    const wailsApp = window.go.main.App
    
    let res = ''
    if (appStore.queryType === 'nfse') {
      res = await wailsApp.QueryNFSe({
        CNPJ: appStore.queryForm.cnpj,
        ChaveAcesso: appStore.queryForm.chave,
      })
    } else {
      res = await wailsApp.QueryNFSeEvents({
        CNPJ: appStore.queryForm.cnpj,
        ChaveAcesso: appStore.queryForm.chave,
      })
    }
    appStore.queryResult = res
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro na consulta: ' + String(err) })
  } finally {
    loading.value = false
  }
}

async function copyResult() {
  if (!appStore.queryResult) return
  await navigator.clipboard.writeText(appStore.queryResult)
  $q.notify({ type: 'positive', message: 'JSON copiado!' })
}
</script>
