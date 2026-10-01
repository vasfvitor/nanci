<template>
  <EventsDialogFrame
    v-model="open"
    title="Eventos da NFS-e"
    :chave-acesso="chaveAcesso"
    :rows="rows"
    :columns="columns"
    :loading="loading"
    :load="load"
  >
    <template #body-cell-tipo="cellProps">
      <q-td :props="cellProps">
        <q-badge
          v-bind="badgeProps(nfseEvent.color(cellProps.row.Type), $q.dark.isActive)"
          :label="nfseEvent.label(cellProps.row.Type)"
        />
      </q-td>
    </template>

    <template #body-cell-descricao="cellProps">
      <q-td :props="cellProps" class="event-wrap">
        {{ cellProps.value }}
      </q-td>
    </template>

    <template #body-cell-substituta="cellProps">
      <q-td :props="cellProps">
        <span class="text-mono" :title="cellProps.row.ReplacementChaveAcesso">
          {{ formatChaveAcesso(cellProps.row.ReplacementChaveAcesso) || '—' }}
        </span>
      </q-td>
    </template>

    <template #body-cell-acoes="cellProps">
      <q-td :props="cellProps">
        <q-btn
          dense
          flat
          round
          size="sm"
          color="grey-7"
          icon="content_copy"
          title="Copiar caminho do XML do evento"
          aria-label="Copiar caminho do XML do evento"
          :disable="!cellProps.row.RawXMLPath"
          @click="copyXMLPath(cellProps.row.RawXMLPath)"
        />
      </q-td>
    </template>
  </EventsDialogFrame>
</template>

<script setup lang="ts">
import { copyToClipboard, useQuasar, type QTableColumn } from 'quasar'
import EventsDialogFrame, { eventDateColumn } from './EventsDialogFrame.vue'
import { useEventList } from '@/composables/useEventList'
import { useNotify } from '@/composables/useNotify'
import { desktopClient } from '@/platform/wails/client'
import type { DocumentEvent } from '@/types/desktop'
import { formatChaveAcesso } from '@/utils/formatters'
import { nfseEvent } from '@/utils/nfseDisplay'
import { badgeProps } from '@/utils/sefazDisplay'

const open = defineModel<boolean>({ required: true })

// documentId loads the events; chaveAcesso is the key the frame shows.
const props = defineProps<{
  documentId: string
  chaveAcesso: string
}>()

const $q = useQuasar()
const { rows, loading, load } = useEventList(() => desktopClient.listEventsForDocument(props.documentId))
const { notifyError, notifyInfo } = useNotify()

const columns: QTableColumn<DocumentEvent>[] = [
  eventDateColumn,
  { name: 'tipo', label: 'Tipo', field: 'Type', align: 'left' },
  {
    name: 'descricao',
    label: 'Descrição',
    field: (row: DocumentEvent) => row.Description || '—',
    align: 'left',
  },
  { name: 'substituta', label: 'Chave substituta', field: 'ReplacementChaveAcesso', align: 'left' },
  { name: 'acoes', label: 'XML', field: () => '', align: 'center' },
]

async function copyXMLPath(path: string) {
  try {
    await copyToClipboard(path)
    notifyInfo('Caminho do XML do evento copiado.', { timeout: 1500 })
  } catch (error) {
    notifyError('Erro ao copiar o caminho do XML', error)
  }
}
</script>
