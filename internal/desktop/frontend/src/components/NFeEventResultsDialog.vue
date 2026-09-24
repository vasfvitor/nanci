<template>
  <q-dialog ref="dialogRef" @hide="onDialogHide">
    <q-card class="q-dialog-plugin nfe-results-dialog">
      <q-card-section>
        <div class="text-h6">Resultado da ciência</div>
        <div class="text-caption text-app-muted">
          {{ counts.registrada }} registradas · {{ counts.ja_registrada }} já registradas ·
          {{ counts.rejeitada }} rejeitadas · {{ counts.nao_enviada }} não enviadas
        </div>
      </q-card-section>

      <q-card-section v-if="result.Interrupted" class="q-pt-none">
        <q-banner dense rounded :class="$q.dark.isActive ? 'bg-grey-9 text-orange-3' : 'bg-orange-1 text-orange-10'">
          <template #avatar>
            <q-icon name="warning" />
          </template>
          O envio foi interrompido: {{ result.Interrupted }}. As notas não enviadas podem ser enviadas
          novamente depois.
        </q-banner>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-table
          :rows="problemResults"
          :columns="columns"
          row-key="ChaveAcesso"
          flat
          bordered
          dense
          :pagination="{ rowsPerPage: 0 }"
          hide-pagination
          no-data-label="Nenhuma nota com problema."
        >
          <template #body-cell-status="cellProps">
            <q-td :props="cellProps">
              <q-badge
                :color="badgeColor(outcomeColor(cellProps.row.Status), $q.dark.isActive)"
                :text-color="badgeTextColor(outcomeColor(cellProps.row.Status), $q.dark.isActive)"
                :label="outcomeLabel(cellProps.row.Status)"
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
import type { NFeEventBatchResult, NFeEventResult } from '@/types/desktop'
import { formatChaveNFe } from '@/utils/formatters'
import { badgeColor, badgeTextColor, outcomeColor, outcomeLabel } from '@/utils/nfeDisplay'
import { countOutcomes } from '@/utils/nfeManifestacao'

const props = defineProps<{
  result: NFeEventBatchResult
}>()

defineEmits<{
  ok: []
  hide: []
}>()

const $q = useQuasar()
const { dialogRef, onDialogHide, onDialogOK } = useDialogPluginComponent()

const counts = computed(() => countOutcomes(props.result.Results))

const problemResults = computed(() =>
  props.result.Results.filter(
    (item) => item.Status !== 'registrada' && item.Status !== 'ja_registrada'
  )
)

const columns: QTableColumn<NFeEventResult>[] = [
  {
    name: 'chave',
    label: 'Chave de acesso',
    field: 'ChaveAcesso',
    align: 'left',
    classes: 'text-mono',
    format: (value: string) => formatChaveNFe(value),
  },
  { name: 'status', label: 'Resultado', field: 'Status', align: 'left' },
  { name: 'cstat', label: 'cStat', field: 'CStat', align: 'left', classes: 'text-mono' },
  { name: 'motivo', label: 'Motivo', field: 'XMotivo', align: 'left' },
]

defineExpose({ dialogRef })
</script>

<style scoped>
.nfe-results-dialog {
  width: 760px;
  max-width: 90vw;
}
</style>
