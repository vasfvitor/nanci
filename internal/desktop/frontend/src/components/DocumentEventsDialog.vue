<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 600px; max-width: 80vw;">
      <q-card-section class="row items-center q-pb-none">
        <div class="text-h6">Histórico de Eventos</div>
        <q-space />
        <q-btn v-close-popup icon="close" flat round dense />
      </q-card-section>

      <q-card-section>
        <div class="text-caption q-mb-md">
          Eventos vinculados ao documento (Cancelamentos, Substituições, etc).
        </div>

        <q-table
          :rows="events"
          :columns="columns"
          row-key="ID"
          :loading="loading"
          flat
          bordered
          dense
          no-data-label="Nenhum evento encontrado."
        >
          <template #body-cell-type="cellProps">
            <q-td :props="cellProps">
              <q-badge :color="cellProps.row.Type === 'cancelamento' ? 'negative' : (cellProps.row.Type === 'substituicao' ? 'warning' : 'grey')">
                {{ cellProps.row.Type.toUpperCase() }}
              </q-badge>
            </q-td>
          </template>
          
          <template #body-cell-acoes="cellProps">
            <q-td :props="cellProps" class="q-gutter-x-sm">
              <q-btn
                dense
                flat
                round
                color="primary"
                icon="code"
                title="Copiar Caminho do XML do Evento"
                @click="copyXML(cellProps.row.RawXMLPath)"
              />
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
import { useQuasar } from 'quasar'
import { useDocuments } from '@/composables/useDocuments'
import type { DocumentEvent } from '@/types/desktop'

const props = defineProps<{
  modelValue: boolean
  documentId: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const $q = useQuasar()
const { loadEvents: loadDocumentEvents } = useDocuments()
const isOpen = ref(props.modelValue)
const loading = ref(false)
const events = ref<DocumentEvent[]>([])

const columns = [
  { name: 'eventAt', label: 'Data', field: (row: DocumentEvent) => row.EventAt ? row.EventAt : '—', align: 'left' as const, sortable: true },
  { name: 'type', label: 'Tipo', field: 'Type', align: 'left' as const, sortable: true },
  { name: 'description', label: 'Descrição / Motivo', field: 'Description', align: 'left' as const },
  { name: 'replacement', label: 'Chave Substituta', field: 'ReplacementChaveAcesso', align: 'left' as const },
  { name: 'acoes', label: 'Ações', field: () => '', align: 'right' as const },
]

watch(() => props.modelValue, (newVal) => {
  isOpen.value = newVal
  if (newVal && props.documentId) {
    loadEvents()
  }
})

watch(isOpen, (newVal) => {
  emit('update:modelValue', newVal)
})

async function loadEvents() {
  events.value = []
  loading.value = true
  try {
    const res = await loadDocumentEvents(props.documentId)
    events.value = res || []
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar eventos: ' + String(err) })
  } finally {
    loading.value = false
  }
}

async function copyXML(xmlPath: string) {
  await navigator.clipboard.writeText(xmlPath)
  $q.notify({ type: 'info', message: 'Caminho do XML copiado para a área de transferência.' })
}
</script>
