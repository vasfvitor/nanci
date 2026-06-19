<template>
  <q-page padding class="column">
    <div class="row items-center q-mb-md">
      <div class="text-h5 text-weight-bold">Consulta Direta (API ADN)</div>
      <q-space />
    </div>

    <q-card flat bordered class="q-pa-md q-mb-md">
      <q-form class="q-gutter-md" @submit="runQuery">
        <div class="row q-col-gutter-md">
          <div class="col-12 col-md-6">
            <q-select
              v-model="form.cnpj"
              :options="companyOptions"
              use-input
              input-debounce="0"
              emit-value
              map-options
              label="Empresa / CNPJ (Para autenticação)"
              outlined
              dense
              :rules="[val => !!val || 'CNPJ é obrigatório']"
              @new-value="createValue"
              @filter="filterFn"
            />
          </div>
          <div class="col-12 col-md-6">
            <q-input
              v-model="form.chave"
              label="Chave de Acesso (50 posições)"
              outlined
              dense
              :rules="[val => !!val || 'Chave é obrigatória', val => /^\d{50}$/.test(val) || 'Chave deve ter 50 números']"
            />
          </div>
        </div>
        <div class="row q-gutter-sm">
          <q-btn type="submit" color="primary" icon="search" label="Consultar NFSe" :loading="loading" @click="type = 'nfse'" />
          <q-btn type="submit" color="secondary" icon="event" label="Consultar Eventos" :loading="loading" @click="type = 'events'" />
        </div>
      </q-form>
    </q-card>

    <q-card v-if="result" flat bordered class="col column q-mt-md">
      <q-toolbar class="dense bg-primary text-white">
        <q-toolbar-title class="text-subtitle2">Resultado da Consulta</q-toolbar-title>
        <q-btn flat round dense icon="content_copy" @click="copyResult">
          <q-tooltip>Copiar JSON</q-tooltip>
        </q-btn>
      </q-toolbar>
      <div class="q-pa-md">
        <pre class="query-result">{{ result }}</pre>
      </div>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useQuery } from '@/composables/useQuery'

const $q = useQuasar()
const query = useQuery()
const { form, result, type, loading, companyOptions } = query

onMounted(async () => {
  try {
    await query.loadCompanies()
  } catch (e) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar empresas para consulta: ' + String(e) })
  }
})

function createValue(val: string, done: (item: unknown, mode: 'add' | 'add-unique' | 'toggle') => void) {
  if (val.length > 0) {
    done(val, 'add-unique')
  }
}

function filterFn(val: string, update: (callback: () => void) => void) {
  update(() => {
    query.filterCompanies(val)
  })
}

async function runQuery() {
  try {
    await query.runQuery()
  } catch (err: unknown) {
    $q.notify({ type: 'negative', message: 'Erro na consulta: ' + String(err) })
  }
}

async function copyResult() {
  if (!result.value) return
  await navigator.clipboard.writeText(result.value)
  $q.notify({ type: 'positive', message: 'JSON copiado!' })
}
</script>

<style scoped>
.query-result {
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  word-wrap: break-word;
}
</style>
