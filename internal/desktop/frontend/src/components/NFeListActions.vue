<template>
  <q-btn
    color="primary"
    icon="task_alt"
    :label="`Registrar ciência (${eligibleCount})`"
    :disable="eligibleCount === 0 || planningCiencia || cienciaInFlight"
    :loading="planningCiencia"
    dense
    flat
    @click="emit('ciencia')"
  />
  <q-btn
    color="primary"
    icon="upload_file"
    label="Importar XML"
    title="Importa XMLs que a empresa já tem, sem consultar a SEFAZ"
    :disable="importDisabled"
    dense
    flat
    @click="emit('import')"
  />
  <q-btn
    color="primary"
    icon="travel_explore"
    label="Buscar pela chave"
    title="Pede uma NF-e à SEFAZ pela chave de acesso"
    :disable="importDisabled"
    dense
    flat
    @click="emit('fetch')"
  />
</template>

<script setup lang="ts">
// NFeListActions are the NF-e buttons of the filter bar (its #actions slot):
// ciência for the eligible selection, XML import and fetch by chave. They
// render without a wrapper, so the bar spaces them like its own buttons.
defineProps<{
  // eligibleCount is how many selected notes can receive ciência.
  eligibleCount: number
  planningCiencia: boolean
  cienciaInFlight: boolean
  // importDisabled disables both the import and the fetch.
  importDisabled: boolean
}>()

const emit = defineEmits<{
  ciencia: []
  import: []
  fetch: []
}>()
</script>
