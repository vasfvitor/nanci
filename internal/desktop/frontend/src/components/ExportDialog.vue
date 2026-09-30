<template>
  <q-dialog ref="dialogRef" @hide="onDialogHide">
    <q-card class="q-dialog-plugin export-dialog">
      <q-card-section>
        <div class="text-h6">Exportar {{ noun }}</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <div class="q-mb-md">{{ scopeText }}</div>

        <template v-if="formats.length > 1">
          <div class="text-subtitle2 q-mb-xs">Formato</div>
          <q-option-group v-model="format" :options="formats" color="primary" class="q-mb-sm" />
        </template>
        <div v-else class="q-mb-sm">
          Formato: <span class="text-weight-medium">{{ formats[0]?.label }}</span>
        </div>

        <div class="column">
          <q-checkbox v-model="incremental" label="Somente não exportados" color="primary" />
          <q-checkbox
            v-if="showIncludeResumos"
            v-model="includeResumos"
            label="Incluir resumos"
            color="primary"
          />
        </div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat color="primary" label="Cancelar" @click="onDialogCancel" />
        <q-btn color="primary" label="Exportar" @click="onOKClick" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script lang="ts">
// ExportChoice is what the user picked in the ExportDialog. The page passes it
// to its export composable; the dialog never calls the backend.
export type ExportChoice = {
  format: string
  incremental: boolean
  includeResumos: boolean
}

export type ExportNoun = 'NFS-e' | 'NF-e' | 'CT-e'

export type ExportScope = 'selected' | 'listed'

// exportScopeText says what an export takes: "Serão exportados 3 CT-e
// selecionados.", "Serão exportadas as 12 NF-e da lista.". NFS-e and NF-e
// are notas (feminine), CT-e is a conhecimento (masculine).
export function exportScopeText(noun: ExportNoun, count: number, scope: ExportScope) {
  const feminine = noun !== 'CT-e'
  const ending = (feminine ? 'a' : 'o') + (count === 1 ? '' : 's')
  const verb = `${count === 1 ? 'Será' : 'Serão'} exportad${ending}`
  if (scope === 'selected') return `${verb} ${count} ${noun} selecionad${ending}.`
  if (count === 1) return `${verb} ${feminine ? 'a' : 'o'} ${noun} da lista.`
  return `${verb} ${feminine ? 'as' : 'os'} ${count} ${noun} da lista.`
}
</script>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useDialogPluginComponent } from 'quasar'

// ExportDialog asks how to export the documents of a page: the format (plain
// text when there is only one), "Somente não exportados" and, when asked,
// "Incluir resumos". The page opens it with $q.dialog and gets an
// ExportChoice in onOk.
const props = withDefaults(
  defineProps<{
    noun: ExportNoun
    count: number
    scope: ExportScope
    formats: { label: string; value: string }[]
    showIncludeResumos?: boolean
    defaultIncremental?: boolean
  }>(),
  {
    showIncludeResumos: false,
    defaultIncremental: false,
  }
)

defineEmits<{
  ok: [choice: ExportChoice]
  hide: []
}>()

const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent()

const format = ref(props.formats[0]?.value ?? '')
const incremental = ref(props.defaultIncremental)
const includeResumos = ref(false)

const scopeText = computed(() => exportScopeText(props.noun, props.count, props.scope))

function onOKClick() {
  const choice: ExportChoice = {
    format: format.value,
    incremental: incremental.value,
    includeResumos: props.showIncludeResumos && includeResumos.value,
  }
  onDialogOK(choice)
}

defineExpose({ dialogRef })
</script>

<style scoped>
.export-dialog {
  min-width: 360px;
}
</style>
