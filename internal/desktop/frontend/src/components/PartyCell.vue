<template>
  <div :title="title">
    <div class="text-weight-medium ellipsis party-name">{{ name || '—' }}</div>
    <div class="text-caption text-app-muted text-mono party-cnpj">
      {{ formatCpfCnpj(cnpj) || '—' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatCpfCnpj } from '@/utils/formatters'

// PartyCell shows a party of a document: the name, cut to the column width,
// over the formatted CPF/CNPJ. The title has both in full.
const props = defineProps<{
  name: string
  cnpj: string
}>()

const title = computed(() => [props.name, formatCpfCnpj(props.cnpj)].filter(Boolean).join('\n'))
</script>

<style scoped>
/* The formatted CNPJ sets the width of the party columns: its letter spacing
   is dropped to keep it narrow, and the name above it is cut to about the
   same width. */
.party-name {
  max-width: 115px;
}

.party-cnpj {
  letter-spacing: normal;
}
</style>
