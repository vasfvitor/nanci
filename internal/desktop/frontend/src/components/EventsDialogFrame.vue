<template>
  <q-dialog v-model="open">
    <q-card class="events-dialog">
      <q-card-section class="row items-center no-wrap q-pb-none">
        <div>
          <div class="text-h6">{{ title }}</div>
          <div class="text-caption text-app-muted text-mono">{{ formatChaveDFe(chaveAcesso) }}</div>
        </div>
        <q-space />
        <q-btn v-close-popup icon="close" flat round dense aria-label="Fechar" />
      </q-card-section>

      <q-card-section>
        <q-table
          :rows="rows"
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
          <template v-for="name in cellSlots" #[name]="cellProps">
            <slot :name="name" v-bind="cellProps" />
          </template>
        </q-table>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn v-close-popup flat label="Fechar" color="primary" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script lang="ts">
import type { ISODateValue } from '@/types/desktop'
import { formatChaveDFe, formatDateTime } from '@/utils/formatters'

// eventDateColumn is the "Data" column every events table starts with.
export const eventDateColumn = {
  name: 'eventAt',
  label: 'Data',
  field: (row: { EventAt?: ISODateValue }) => row.EventAt,
  align: 'left' as const,
  sortable: true,
  format: (value: ISODateValue) => formatDateTime(value ?? null, '—'),
}
</script>

<script setup lang="ts" generic="Row extends { ID: string }">
import { useSlots, watch } from 'vue'
import type { QTableColumn } from 'quasar'
import { useNotify } from '@/composables/useNotify'

// EventsDialogFrame is the events dialog of the document pages: the title,
// the access key grouped as a subtitle, the events table and a "Fechar"
// button. Each opening calls load; the dialog passes its columns and its
// custom cells as body-cell-<column> slots.
const props = defineProps<{
  title: string
  chaveAcesso: string
  rows: Row[]
  columns: QTableColumn<Row>[]
  loading: boolean
  load: () => Promise<void>
}>()

const open = defineModel<boolean>({ required: true })

const cellSlots = Object.keys(useSlots()).filter((name) => name.startsWith('body-cell-'))

const { notifyError } = useNotify()

watch(open, async (isOpen) => {
  if (!isOpen || !props.chaveAcesso) return
  try {
    await props.load()
  } catch (error) {
    notifyError('Erro ao carregar eventos', error)
  }
})
</script>

<style scoped>
.events-dialog {
  min-width: 760px;
  max-width: 90vw;
}
</style>
