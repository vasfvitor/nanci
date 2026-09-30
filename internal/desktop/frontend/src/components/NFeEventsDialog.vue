<template>
  <EventsDialogFrame v-model="open" title="Eventos da NF-e" :chave-acesso="chaveAcesso">
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
      <template #body-cell-tipo="cellProps">
        <q-td :props="cellProps">
          <q-badge
            v-bind="badgeProps(nfeEventColor(cellProps.row.TpEvento), $q.dark.isActive)"
            :label="nfeEventLabel(cellProps.row.TpEvento)"
          />
          <div class="text-caption text-app-muted text-mono">{{ cellProps.row.TpEvento }}</div>
        </q-td>
      </template>

      <template #body-cell-status="cellProps">
        <q-td :props="cellProps">
          <span class="text-mono">{{ cellProps.row.CStat || '—' }}</span>
          <div class="text-caption">{{ cellProps.row.XMotivo }}</div>
        </q-td>
      </template>
    </q-table>
  </EventsDialogFrame>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import EventsDialogFrame from './EventsDialogFrame.vue'
import { useNFeEvents } from '@/composables/useNFeEvents'
import { useNotify } from '@/composables/useNotify'
import type { NFeEvent } from '@/types/desktop'
import { formatDateTime } from '@/utils/formatters'
import { nfeEventColor, nfeEventLabel } from '@/utils/nfeDisplay'
import { badgeProps } from '@/utils/sefazDisplay'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  cnpj: string
  chaveAcesso: string
}>()

const $q = useQuasar()
const { events, loading, load } = useNFeEvents()
const { notifyError } = useNotify()

const columns: QTableColumn<NFeEvent>[] = [
  {
    name: 'eventAt',
    label: 'Data',
    field: 'EventAt',
    align: 'left',
    sortable: true,
    format: (value: NFeEvent['EventAt']) => formatDateTime(value ?? null, '—'),
  },
  { name: 'tipo', label: 'Tipo', field: 'TpEvento', align: 'left' },
  { name: 'seq', label: 'Seq.', field: 'NSeqEvento', align: 'right' },
  {
    name: 'protocolo',
    label: 'Protocolo',
    field: (row: NFeEvent) => row.Protocolo || '—',
    align: 'left',
    classes: 'text-mono',
  },
  { name: 'status', label: 'cStat / Motivo', field: 'CStat', align: 'left' },
  { name: 'detalhe', label: 'Detalhe', field: eventDetail, align: 'left' },
  {
    name: 'origem',
    label: 'Origem',
    field: (row: NFeEvent) => (row.SentByNanci ? 'Enviado pelo Nanci' : 'Distribuição DF-e'),
    align: 'left',
  },
]

function eventDetail(row: NFeEvent) {
  return row.Justificativa || row.Correcao || row.Description || '—'
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

