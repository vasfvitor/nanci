<template>
  <q-page padding class="column">
    <div class="row items-center q-mb-md">
      <div class="text-h5 text-weight-bold">Consulta Direta (API ADN)</div>
      <q-space />
    </div>

    <q-banner
      v-if="loaded && !cnpj"
      dense
      rounded
      class="q-mb-md"
      :class="$q.dark.isActive ? 'bg-grey-10' : 'bg-grey-1'"
    >
      <template #avatar>
        <q-icon name="info" color="primary" />
      </template>
      Nenhuma empresa cadastrada. A consulta autentica com a empresa escolhida no menu lateral.
      <template #action>
        <q-btn flat dense color="primary" label="Cadastrar empresa" to="/" />
      </template>
    </q-banner>

    <q-card flat bordered class="q-pa-md q-mb-md">
      <q-form class="q-gutter-md" @submit="runQuery">
        <div class="row q-col-gutter-md">
          <div class="col-12 col-md-6">
            <q-input
              :model-value="companyLabel"
              label="Empresa (autenticação)"
              hint="Troque no menu lateral"
              readonly
              outlined
              dense
            />
          </div>
          <div class="col-12 col-md-6">
            <q-select
              v-model="form.chave"
              :options="documentOptions"
              use-input
              fill-input
              hide-selected
              input-debounce="0"
              emit-value
              map-options
              label="Chave de Acesso (50 posições)"
              outlined
              dense
              :rules="[
                val => !!val || 'Chave é obrigatória',
                val => {
                  const v = typeof val === 'object' && val !== null ? val.value : val;
                  return isChaveNFSe(v) || 'Chave deve ter 50 posições; letras só no CNPJ do emitente';
                }
              ]"
              @new-value="createValue"
              @filter="filterDocumentsFn"
              @input-value="(v) => { if (v && v.length === 50) form.chave = v }"
            >
              <template #option="scope">
                <q-item v-bind="scope.itemProps">
                  <q-item-section>
                    <q-item-label>{{ scope.opt.description }}</q-item-label>
                  </q-item-section>
                </q-item>
              </template>
              <template #no-option>
                <q-item>
                  <q-item-section class="text-grey">
                    Digite a chave completa ou sincronize notas primeiro.
                  </q-item-section>
                </q-item>
              </template>
            </q-select>
          </div>
        </div>
        <div class="row q-gutter-sm">
          <q-btn
            type="submit"
            color="primary"
            icon="search"
            label="Consultar no ADN"
            :loading="loading"
            :disable="!cnpj"
          />
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
import { computed } from 'vue'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import { errorMessage } from '@/platform/wails/client'
import { companyOption } from '@/composables/useCompanies'
import { useQuery } from '@/composables/useQuery'
import { useWorkspaceStore } from '@/stores/workspace'
import { isChaveNFSe } from '@/utils/formatters'

const $q = useQuasar()
const query = useQuery()
const { form, result, loading, documentOptions } = query
// The query authenticates as the workspace company.
const { cnpj, selectedCompany, loaded } = storeToRefs(useWorkspaceStore())

// companyLabel names the workspace company the query authenticates as.
const companyLabel = computed(() =>
  selectedCompany.value ? companyOption(selectedCompany.value).label : ''
)

function createValue(val: string, done: (item: unknown, mode: 'add' | 'add-unique' | 'toggle') => void) {
  if (val.length > 0) {
    done(val, 'add-unique')
  }
}

function filterDocumentsFn(val: string, update: (callback: () => void) => void) {
  update(() => {
    query.filterDocuments(val)
  })
}

async function runQuery() {
  try {
    await query.runQuery()
  } catch (err: unknown) {
    $q.notify({ type: 'negative', message: 'Erro na consulta: ' + errorMessage(err) })
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
