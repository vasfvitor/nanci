<template>
  <RowActionsMenu v-model:expanded="expanded" source="nfe">
    <RowMenuItem
      label="Manifestar…"
      :caption="manifestarCaption"
      :disable="!canManifest"
      @click="emit('manifest')"
    />
    <RowMenuItem
      label="Registrar ciência"
      :caption="cienciaCaption"
      :disable="!canRegisterCiencia"
      @click="emit('ciencia')"
    />
    <RowMenuItem
      :label="`Eventos (${row.EventCount})`"
      :disable="row.EventCount === 0"
      @click="emit('events')"
    />
    <RowMenuItem
      label="Exportar XML"
      :caption="exportCaption"
      :disable="exporting || !actions.canExportXML"
      @click="emit('exportXml')"
    />
    <RowMenuItem
      label="Exportar DANFE"
      :caption="exportCaption"
      :disable="exporting || !actions.canExportXML"
      @click="emit('exportDanfe')"
    />
  </RowActionsMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import RowActionsMenu from './RowActionsMenu.vue'
import RowMenuItem from './RowMenuItem.vue'
import type { NFeRow } from '@/types/desktop'
import { nfeRowActions } from '@/utils/nfeManifestacao'

// NFeRowMenu is the "Ações" cell of an NF-e row. An entry the note cannot
// take is disabled and says why in its caption.
const props = defineProps<{
  row: NFeRow
  // busy is true while an event of this note is being sent.
  busy: boolean
  // cienciaInFlight is true while any ciência is being sent.
  cienciaInFlight: boolean
  exporting: boolean
}>()

const emit = defineEmits<{
  manifest: []
  ciencia: []
  events: []
  exportXml: []
  exportDanfe: []
}>()

const expanded = defineModel<boolean>('expanded', { default: false })

// A note with an event being sent takes no other; its menu says why.
const SENDING_CAPTION = 'Envio em andamento'

const actions = computed(() => nfeRowActions(props.row))

const canManifest = computed(() => actions.value.conclusiveBlockReason === null && !props.busy)

const manifestarCaption = computed(() =>
  props.busy ? SENDING_CAPTION : (actions.value.conclusiveBlockReason ?? '')
)

const canRegisterCiencia = computed(
  () => actions.value.cienciaBlockReason === null && !props.cienciaInFlight && !props.busy
)

const cienciaCaption = computed(() =>
  props.busy ? SENDING_CAPTION : (actions.value.cienciaBlockReason ?? '')
)

const exportCaption = computed(() =>
  actions.value.canExportXML ? '' : 'XML completo ainda não baixado'
)
</script>
