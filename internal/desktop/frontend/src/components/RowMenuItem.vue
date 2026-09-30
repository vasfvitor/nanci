<template>
  <q-item clickable :disable="disable" @click="onClick">
    <q-item-section>
      <q-item-label>{{ label }}</q-item-label>
      <q-item-label v-if="caption" caption>{{ caption }}</q-item-label>
    </q-item-section>
  </q-item>
</template>

<script setup lang="ts">
// RowMenuItem is one entry of a RowActionsMenu. A disabled entry can say why
// in its caption.
const props = defineProps<{
  label: string
  caption?: string
  disable?: boolean
}>()

const emit = defineEmits<{
  click: []
}>()

function onClick() {
  if (!props.disable) emit('click')
}
</script>
