<template>
  <q-dialog ref="dialogRef" @hide="onDialogHide">
    <q-card class="q-dialog-plugin import-result-dialog">
      <q-card-section>
        <div class="text-h6">{{ title }}</div>
        <div class="text-caption text-app-muted">{{ summary }}</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-table
          :rows="result.Files"
          :columns="columns"
          row-key="Path"
          flat
          bordered
          dense
          wrap-cells
          :pagination="{ rowsPerPage: 0 }"
          hide-pagination
          no-data-label="Nenhum arquivo importado."
        >
          <template #body-cell-arquivo="cellProps">
            <q-td :props="cellProps" :title="cellProps.row.Path">{{ cellProps.value }}</q-td>
          </template>
          <template #body-cell-outcome="cellProps">
            <q-td :props="cellProps">
              <q-badge
                v-bind="badgeProps(importOutcomeColor(cellProps.row.Outcome), $q.dark.isActive)"
                :label="importOutcomeLabel(cellProps.row.Outcome)"
              />
            </q-td>
          </template>
        </q-table>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn color="primary" flat label="Fechar" @click="onDialogOK()" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useDialogPluginComponent, useQuasar, type QTableColumn } from 'quasar'
import type { ImportedFile, ImportXMLResult } from '@/types/desktop'
import { DOCUMENT_SOURCES } from '@/utils/documentSources'
import { formatChaveDFe } from '@/utils/formatters'
import {
  fileName,
  importKindLabel,
  importOutcomeColor,
  importOutcomeLabel,
  importSummary,
} from '@/utils/importDisplay'
import { badgeProps } from '@/utils/sefazDisplay'

const props = defineProps<{
  result: ImportXMLResult
  source: 'nfe' | 'cte'
}>()

defineEmits<{
  ok: []
  hide: []
}>()

const $q = useQuasar()
const { dialogRef, onDialogHide, onDialogOK } = useDialogPluginComponent()

const title = computed(() => {
  const { article, noun } = DOCUMENT_SOURCES[props.source]
  return `Importação de XML ${article} ${noun}`
})

const summary = computed(() => importSummary(props.result))

const columns: QTableColumn<ImportedFile>[] = [
  {
    name: 'arquivo',
    label: 'Arquivo',
    field: 'Path',
    align: 'left',
    format: (value: string) => fileName(value),
  },
  { name: 'outcome', label: 'Resultado', field: 'Outcome', align: 'left' },
  {
    name: 'kind',
    label: 'Tipo',
    field: 'Kind',
    align: 'left',
    format: (value: string) => importKindLabel(value),
  },
  {
    name: 'chave',
    label: 'Chave de acesso',
    field: 'ChaveAcesso',
    align: 'left',
    classes: 'text-mono',
    format: (value: string) => formatChaveDFe(value),
  },
  { name: 'message', label: 'Mensagem', field: 'Message', align: 'left' },
]

defineExpose({ dialogRef })
</script>

<style scoped>
.import-result-dialog {
  width: 960px;
  max-width: 92vw;
}
</style>
