<template>
  <div v-if="warnings.length > 0">
    <div class="text-subtitle2 text-primary q-mb-xs">Avisos da leitura do XML</div>
    <ul class="q-my-none q-pl-md text-body2">
      <li v-for="warning in warnings" :key="warning">{{ warning }}</li>
    </ul>
  </div>
</template>

<script setup lang="ts">
// ParseWarnings lists what the parser noticed in a document's XML, in the
// expanded details. It renders nothing when there is no warning.
defineProps<{
  warnings: string[]
}>()
</script>
