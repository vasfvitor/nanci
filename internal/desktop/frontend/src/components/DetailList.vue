<template>
  <div>
    <div class="text-subtitle2 text-primary q-mb-xs">{{ title }}</div>
    <dl class="detail-list text-body2">
      <template v-for="item in items" :key="item.label">
        <dt>{{ item.label }}</dt>
        <dd>
          <span :class="{ 'text-mono': item.mono }">{{ displayValue(item.value) }}</span>
          <span v-if="item.caption" class="text-caption text-app-muted text-mono q-ml-xs">
            {{ item.caption }}
          </span>
        </dd>
      </template>
    </dl>
  </div>
</template>

<script lang="ts">
// DetailItem is one line of a DetailList. caption is an optional muted note
// after the value, such as the CNPJ after a name.
export type DetailItem = {
  label: string
  value: string | number | null | undefined
  mono?: boolean
  caption?: string
}
</script>

<script setup lang="ts">
// DetailList is a titled group of label/value lines in the expanded details
// of a document row. An empty value is shown as "—".
defineProps<{
  title: string
  items: DetailItem[]
}>()

function displayValue(value: DetailItem['value']) {
  if (value === null || value === undefined || value === '') return '—'
  return String(value)
}
</script>

<style scoped>
.detail-list {
  display: grid;
  grid-template-columns: max-content 1fr;
  column-gap: 12px;
  row-gap: 2px;
  margin: 0;
}

.detail-list dt {
  font-weight: 500;
}

.detail-list dd {
  margin: 0;
}
</style>
