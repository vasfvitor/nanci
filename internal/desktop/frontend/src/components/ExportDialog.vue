<template>
  <q-dialog ref="dialogRef" @hide="onDialogHide">
    <q-card class="q-dialog-plugin" style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">{{ selectedCount > 0 ? 'Exportar Documentos Selecionados' : 'Exportar Documentos' }}</div>
      </q-card-section>

      <q-card-section>
        <div v-if="selectedCount > 0" class="q-mb-md">
          Deseja exportar os {{ selectedCount }} documentos selecionados?
        </div>
        <div v-else class="q-mb-md">
          Todos os documentos ou apenas os que não foram exportados:
        </div>

        <div class="text-subtitle2 q-mb-sm">Formato de Exportação:</div>
        <q-option-group
          v-model="format"
          :options="formatOptions"
          color="primary"
        />

        <div v-if="selectedCount === 0" class="q-mt-md">
          <q-checkbox v-model="incremental" label="Exportar somente novos" color="primary" />
        </div>
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="Cancelar" @click="onDialogCancel" />
        <q-btn color="primary" label="Exportar" @click="onOKClick" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useDialogPluginComponent } from 'quasar'

defineProps<{
  selectedCount: number
}>()

defineEmits<{
  ok: [payload: { format: string, incremental: boolean }]
  hide: []
}>()

const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent()

const format = ref('csv')
const incremental = ref(false)

const formatOptions = [
  { label: 'Planilha CSV', value: 'csv' },
  { label: 'Planilha Excel (XLSX)', value: 'xlsx' },
  { label: 'XMLs Originais (ZIP)', value: 'zip' },
  { label: 'DANFSes (ZIP)', value: 'danfse-zip' }
]

function onOKClick() {
  onDialogOK({
    format: format.value,
    incremental: incremental.value
  })
}

defineExpose({ dialogRef })
</script>
