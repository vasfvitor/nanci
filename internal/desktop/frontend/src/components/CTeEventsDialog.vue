<template>
  <q-dialog v-model="open">
    <q-card class="cte-events-dialog">
      <q-card-section class="row items-center q-pb-none">
        <div>
          <div class="text-h6">Eventos do CT-e</div>
          <div class="text-caption text-app-muted text-mono">{{ formatChaveDFe(chaveAcesso) }}</div>
        </div>
        <q-space />
        <q-btn v-close-popup icon="close" flat round dense aria-label="Fechar" />
      </q-card-section>

      <q-card-section>
        <q-table
          :rows="events"
          :columns="columns"
          row-key="ID"
          :loading="loading"
          flat
          bordered
          dense
          :pagination="{ rowsPerPage: 0 }"
          hide-pagination
          no-data-label="Nenhum evento encontrado."
        >
          <template #body-cell-eventAt="cellProps">
            <q-td :props="cellProps">
              {{ cellProps.value }}
              <div v-if="cellProps.row.RegisteredAt" class="text-caption text-app-muted">
                Registro: {{ formatDateTime(cellProps.row.RegisteredAt) }}
              </div>
            </q-td>
          </template>

          <template #body-cell-tipo="cellProps">
            <q-td :props="cellProps">
              <q-badge
                v-bind="badgeProps(cteEventColor(cellProps.row.Type), $q.dark.isActive)"
                :label="cteEventTitle(cellProps.row)"
              />
              <div class="text-caption text-app-muted text-mono">
                {{ cellProps.row.TpEvento }} · seq. {{ cellProps.row.NSeqEvento }}
              </div>
            </q-td>
          </template>

          <template #body-cell-status="cellProps">
            <q-td :props="cellProps" class="cte-event-wrap">
              <span class="text-mono">{{ cellProps.row.CStat || '—' }}</span>
              <div class="text-caption">{{ cellProps.row.XMotivo }}</div>
            </q-td>
          </template>

          <template #body-cell-detalhe="cellProps">
            <q-td :props="cellProps" class="cte-event-wrap">
              <div v-for="line in eventDetails(cellProps.row)" :key="line.label">
                <span class="text-weight-medium">{{ line.label }}:</span> {{ line.text }}
              </div>
              <span v-if="eventDetails(cellProps.row).length === 0">—</span>
            </q-td>
          </template>
        </q-table>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn v-close-popup flat label="Fechar" color="primary" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import { useCTeEvents } from '@/composables/useCTeEvents'
import { useNotify } from '@/composables/useNotify'
import type { CTeEvent } from '@/types/desktop'
import { cteEventColor, cteEventTitle } from '@/utils/cteDisplay'
import { formatChaveDFe, formatCpfCnpj, formatDateTime } from '@/utils/formatters'
import { badgeProps } from '@/utils/sefazDisplay'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  cnpj: string
  chaveAcesso: string
}>()

const $q = useQuasar()
const { events, loading, load } = useCTeEvents()
const { notifyError } = useNotify()

const columns: QTableColumn<CTeEvent>[] = [
  {
    name: 'eventAt',
    label: 'Data',
    field: 'EventAt',
    align: 'left',
    sortable: true,
    format: (value: CTeEvent['EventAt']) => formatDateTime(value ?? null, '—'),
  },
  { name: 'tipo', label: 'Tipo', field: 'TpEvento', align: 'left' },
  {
    name: 'protocolo',
    label: 'Protocolo',
    field: (row: CTeEvent) => row.Protocolo || '—',
    align: 'left',
    classes: 'text-mono',
  },
  { name: 'status', label: 'cStat / Motivo', field: 'CStat', align: 'left' },
  { name: 'detalhe', label: 'Detalhe', field: 'Justificativa', align: 'left' },
  {
    name: 'autor',
    label: 'Autor',
    field: (row: CTeEvent) => formatCpfCnpj(row.AutorCNPJ) || '—',
    align: 'left',
    classes: 'text-mono',
  },
]

// eventDetails lists the free-text fields the event carries.
function eventDetails(row: CTeEvent) {
  return [
    { label: 'Justificativa', text: row.Justificativa },
    { label: 'Correção', text: row.Correcao },
    { label: 'Observação', text: row.Observacao },
  ].filter((line) => line.text)
}

watch(open, (isOpen) => {
  if (isOpen && props.chaveAcesso) {
    void loadEvents()
  }
})

async function loadEvents() {
  try {
    await load(props.cnpj, props.chaveAcesso)
  } catch (error) {
    notifyError('Erro ao carregar eventos', error)
  }
}
</script>

<style scoped>
.cte-events-dialog {
  min-width: 760px;
  max-width: 90vw;
}

.cte-event-wrap {
  min-width: 180px;
  max-width: 260px;
  white-space: normal;
  overflow-wrap: break-word;
}
</style>
