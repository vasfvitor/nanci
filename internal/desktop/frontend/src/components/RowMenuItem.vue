<template>
  <q-item clickable :disable="disable" @click="emit('click')">
    <q-item-section>
      <q-item-label>{{ label }}</q-item-label>
      <q-item-label v-if="caption" caption>{{ caption }}</q-item-label>
    </q-item-section>
  </q-item>
</template>

<script setup lang="ts">
// RowMenuItem is one entry of a RowActionsMenu. A disabled entry can say why
// in its caption; q-item does not emit click while disabled.
defineProps<{
  label: string
  caption?: string
  disable?: boolean
}>()

const emit = defineEmits<{
  click: []
}>()
</script>
