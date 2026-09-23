<template>
  <q-dialog v-model="isOpen">
    <q-card class="nfe-events-dialog">
      <q-card-section class="row items-center q-pb-none">
        <div>
          <div class="text-h6">Eventos da NF-e</div>
          <div class="text-caption text-app-muted text-mono">{{ formatChaveNFe(chaveAcesso) }}</div>
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
          <template #body-cell-tipo="cellProps">
            <q-td :props="cellProps">
              <q-badge
                :color="badgeColor(nfeEventColor(cellProps.row.TpEvento), $q.dark.isActive)"
                :text-color="badgeTextColor(nfeEventColor(cellProps.row.TpEvento), $q.dark.isActive)"
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
      </q-card-section>

      <q-card-actions align="right">
        <q-btn v-close-popup flat label="Fechar" color="primary" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import { useNFeDocuments } from '@/composables/useNFeDocuments'
import type { NFeEvent } from '@/types/desktop'
import { formatChaveNFe, formatDateTime } from '@/utils/formatters'
import { badgeColor, badgeTextColor, nfeEventColor, nfeEventLabel } from '@/utils/nfeDisplay'

const props = defineProps<{
  modelValue: boolean
  cnpj: string
  chaveAcesso: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const $q = useQuasar()
const { loadEvents: loadNFeEvents } = useNFeDocuments()
const isOpen = ref(props.modelValue)
const loading = ref(false)
const events = ref<NFeEvent[]>([])

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
  { name: 'protocolo', label: 'Protocolo', field: 'Protocolo', align: 'left', classes: 'text-mono' },
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

watch(
  () => props.modelValue,
  (newVal) => {
    isOpen.value = newVal
    if (newVal && props.chaveAcesso) {
      void loadEvents()
    }
  }
)

watch(isOpen, (newVal) => {
  emit('update:modelValue', newVal)
})

async function loadEvents() {
  events.value = []
  loading.value = true
  try {
    events.value = await loadNFeEvents(props.chaveAcesso, props.cnpj)
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar eventos: ' + String(err) })
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.nfe-events-dialog {
  min-width: 720px;
  max-width: 90vw;
}
</style>
