<template>
  <div class="row q-gutter-sm items-center q-mb-md q-pa-sm rounded-borders shadow-1">
    <slot :search="search" />

    <q-toggle
      :model-value="onlyUnviewed"
      label="Somente não vistos"
      dense
      :disable="loading"
      @update:model-value="onOnlyUnviewedChange"
    />

    <q-space />

    <div class="row no-wrap items-center q-gutter-sm">
      <q-btn
        color="primary"
        icon="search"
        label="Buscar"
        :disable="!canSearch"
        :loading="loading"
        dense
        flat
        @click="search"
      />
      <slot name="actions" />
      <q-btn
        color="primary"
        icon="done_all"
        :label="`Marcar vistos (${markViewedCount})`"
        :disable="loading || markViewedCount === 0"
        dense
        flat
        @click="emit('markViewed')"
      />
      <q-btn
        color="secondary"
        icon="folder_zip"
        label="Exportar"
        :disable="loading || exporting || exportDisabled"
        :loading="exporting"
        dense
        flat
        @click="emit('export')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

// DocumentFilterBar is the filter row of a document page, in the standard
// order: the source's own filters (default slot), "Somente não vistos",
// then Buscar, the source's actions (#actions), "Marcar vistos (n)" and
// Exportar. The company and competência live in the left drawer. Changing
// the toggle searches; the default slot gets search to bind on its own text
// fields.
const props = withDefaults(
  defineProps<{
    loading: boolean
    exporting: boolean
    exportDisabled: boolean
    searchDisabled?: boolean
    markViewedCount: number
  }>(),
  {
    searchDisabled: false,
  }
)

const onlyUnviewed = defineModel<boolean>('onlyUnviewed', { required: true })

const emit = defineEmits<{
  search: []
  markViewed: []
  export: []
}>()

const canSearch = computed(() => !props.loading && !props.searchDisabled)

// search emits search unless the bar is loading or told not to search.
function search() {
  if (canSearch.value) emit('search')
}

function onOnlyUnviewedChange(value: boolean) {
  onlyUnviewed.value = value
  emit('search')
}
</script>
