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
            <q-input
              v-model="form.cnpj"
              label="CNPJ (Para autenticação)"
              mask="##.###.###/####-##"
              unmasked-value
              outlined
              dense
              :rules="[val => !!val || 'CNPJ é obrigatório']"
            />
          </div>
          <div class="col-12 col-md-6">
            <q-input
              v-model="form.chave"
              label="Chave de Acesso (50 posições)"
              outlined
              dense
              :rules="[val => !!val || 'Chave é obrigatória', val => val.length === 50 || 'Chave deve ter 50 números']"
            />
          </div>
        </div>
        <div class="row q-gutter-sm">
          <q-btn type="submit" color="primary" icon="search" label="Consultar NFSe" :loading="loading" @click="queryType = 'nfse'" />
          <q-btn type="submit" color="secondary" icon="event" label="Consultar Eventos" :loading="loading" @click="queryType = 'events'" />
        </div>
      </q-form>
    </q-card>

    <q-card flat bordered class="col column q-mt-md" v-if="result">
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
import { ref, computed } from 'vue'
import { useQuasar } from 'quasar'

const $q = useQuasar()

const form = ref({
  cnpj: '',
  chave: '',
})

const loading = ref(false)
const queryType = ref('nfse')
const result = ref('')

const highlightedResult = computed(() => {
  if (!result.value) return ''
  let jsonStr = result.value
  jsonStr = jsonStr.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return jsonStr.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, (match) => {
    let cls = 'text-positive' // string default
    if (/^"/.test(match)) {
      if (/:$/.test(match)) {
        cls = 'text-primary text-weight-bold' // key
      }
    } else if (/true|false/.test(match)) {
      cls = 'text-secondary text-weight-bold' // boolean
    } else if (/null/.test(match)) {
      cls = 'text-grey' // null
    } else {
      cls = 'text-accent' // number
    }
    return `<span class="${cls}">${match}</span>`
  })
})

async function runQuery() {
  if (!form.value.cnpj || form.value.chave.length !== 50) return

  loading.value = true
  result.value = ''

  try {
    // We use ts-ignore dynamically calling the Wails backend since bindings might not be generated yet
    // @ts-ignore
    const wailsApp = window.go.main.App
    
    let res = ''
    if (queryType.value === 'nfse') {
      res = await wailsApp.QueryNFSe({
        CNPJ: form.value.cnpj,
        ChaveAcesso: form.value.chave,
      })
    } else {
      res = await wailsApp.QueryNFSeEvents({
        CNPJ: form.value.cnpj,
        ChaveAcesso: form.value.chave,
      })
    }
    result.value = res
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro na consulta: ' + String(err) })
  } finally {
    loading.value = false
  }
}

async function copyResult() {
  if (!result.value) return
  await navigator.clipboard.writeText(result.value)
  $q.notify({ type: 'positive', message: 'JSON copiado!' })
}
</script>
