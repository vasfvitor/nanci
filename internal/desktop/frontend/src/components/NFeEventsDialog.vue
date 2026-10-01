<template>
  <EventsDialogFrame
    v-model="open"
    title="Eventos da NF-e"
    :chave-acesso="chaveAcesso"
    :rows="rows"
    :columns="columns"
    :loading="loading"
    :load="load"
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
  </EventsDialogFrame>
</template>

<script setup lang="ts">
import { useQuasar, type QTableColumn } from 'quasar'
import EventsDialogFrame, { eventDateColumn } from './EventsDialogFrame.vue'
import { useEventList } from '@/composables/useEventList'
import { desktopClient } from '@/platform/wails/client'
import type { NFeEvent } from '@/types/desktop'
import { nfeEventColor, nfeEventLabel } from '@/utils/nfeDisplay'
import { badgeProps } from '@/utils/sefazDisplay'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  cnpj: string
  chaveAcesso: string
}>()

const $q = useQuasar()
const { rows, loading, load } = useEventList(() =>
  desktopClient.listNFeEvents(props.cnpj, props.chaveAcesso)
)

const columns: QTableColumn<NFeEvent>[] = [
  eventDateColumn,
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
</script>
