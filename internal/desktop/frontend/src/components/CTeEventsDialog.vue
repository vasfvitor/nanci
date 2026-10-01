<template>
  <EventsDialogFrame
    v-model="open"
    title="Eventos do CT-e"
    :chave-acesso="chaveAcesso"
    :rows="rows"
    :columns="columns"
    :loading="loading"
    :load="load"
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
      <q-td :props="cellProps" class="event-wrap">
        <span class="text-mono">{{ cellProps.row.CStat || '—' }}</span>
        <div class="text-caption">{{ cellProps.row.XMotivo }}</div>
      </q-td>
    </template>

    <template #body-cell-detalhe="cellProps">
      <q-td :props="cellProps" class="event-wrap">
        <div v-for="line in eventDetails(cellProps.row)" :key="line.label">
          <span class="text-weight-medium">{{ line.label }}:</span> {{ line.text }}
        </div>
        <span v-if="eventDetails(cellProps.row).length === 0">—</span>
      </q-td>
    </template>
  </EventsDialogFrame>
</template>

<script setup lang="ts">
import { useQuasar, type QTableColumn } from 'quasar'
import EventsDialogFrame, { eventDateColumn } from './EventsDialogFrame.vue'
import { useEventList } from '@/composables/useEventList'
import { desktopClient } from '@/platform/wails/client'
import type { CTeEvent } from '@/types/desktop'
import { cteEventColor, cteEventTitle } from '@/utils/cteDisplay'
import { formatCpfCnpj, formatDateTime } from '@/utils/formatters'
import { badgeProps } from '@/utils/sefazDisplay'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  cnpj: string
  chaveAcesso: string
}>()

const $q = useQuasar()
const { rows, loading, load } = useEventList(() =>
  desktopClient.listCTeEvents(props.cnpj, props.chaveAcesso)
)

const columns: QTableColumn<CTeEvent>[] = [
  eventDateColumn,
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
</script>
