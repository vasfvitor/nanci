<template>
  <q-chip
    removable
    outline
    color="primary"
    icon="filter_alt"
    class="q-ml-none q-mb-md"
    :title="`Mostra só ${noun} ${formatChaveDFe(chave)}, de qualquer competência`"
    @remove="emit('clear')"
  >
    Filtrado pela chave {{ formatChaveDFe(chave) }}
  </q-chip>
</template>

<script setup lang="ts">
import { formatChaveDFe } from '@/utils/formatters'

// ChaveFocusChip tells that the page lists one document, opened from a link
// on the other document page; removing it goes back to the filter.
defineProps<{
  chave: string
  // noun names the document with its article, such as "a NF-e".
  noun: string
}>()

const emit = defineEmits<{
  clear: []
}>()
</script>
