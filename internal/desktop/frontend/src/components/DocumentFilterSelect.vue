<template>
  <q-select
    v-model="model"
    class="document-filter-select"
    :options="options"
    :label="label"
    emit-value
    map-options
    outlined
    dense
    options-dense
    :disable="disable"
  />
</template>

<script setup lang="ts">
import type { FilterOption } from '@/utils/sefazDisplay'

// DocumentFilterSelect is one of a source's own filters in the filter bar,
// such as Papel or Situação. Its options send the value the backend filters
// by; an empty value means all.
defineProps<{
  options: FilterOption[]
  label: string
  disable: boolean
}>()

const model = defineModel<string>({ required: true })
</script>

<style scoped>
.document-filter-select {
  min-width: 130px;
}
</style>
