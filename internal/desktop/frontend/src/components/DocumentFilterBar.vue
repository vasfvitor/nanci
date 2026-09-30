<template>
  <div
    class="row q-gutter-sm items-center q-mb-md q-pa-sm rounded-borders shadow-1"
    @keyup.enter="onEnter"
  >
    <q-select
      :model-value="cnpj"
      class="col-12 col-md-3"
      :options="companyOptions"
      label="Empresa"
      emit-value
      map-options
      outlined
      dense
      options-dense
      :disable="loading"
      @update:model-value="onCompanyChange"
    />

    <div class="col-12 col-sm-6 col-md-3" :title="competenceTitle">
      <CompetencePicker v-model="competence" :disable="loading" />
    </div>

    <slot />

    <q-toggle
      v-if="onlyUnviewed !== undefined"
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
        @click="emit('search')"
      />
      <slot name="actions" />
      <q-btn
        v-if="onlyUnviewed !== undefined"
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
import CompetencePicker from './CompetencePicker.vue'

// DocumentFilterBar is the filter row of a document page, in the standard
// order: company, competência, the source's own filters (default slot, with
// the document-filter-select class on selects), "Somente não vistos", then
// Buscar, the source's actions (#actions), "Marcar vistos (n)" and Exportar.
// Enter in a text field searches, but not on a select; changing the company
// or the toggle searches too, via companyChange and search. Leaving onlyUnviewed undefined hides the toggle
// and "Marcar vistos", for sources without viewed marks.
const props = withDefaults(
  defineProps<{
    companyOptions: { label: string; value: string }[]
    loading: boolean
    exporting: boolean
    exportDisabled: boolean
    searchDisabled?: boolean
    competenceTitle?: string
    markViewedCount?: number
  }>(),
  {
    searchDisabled: false,
    competenceTitle: 'Competência pelo mês de emissão',
    markViewedCount: 0,
  }
)

const cnpj = defineModel<string>('cnpj', { required: true })
const competence = defineModel<string>('competence', { required: true })
const onlyUnviewed = defineModel<boolean | undefined>('onlyUnviewed', { default: undefined })

const emit = defineEmits<{
  search: []
  companyChange: []
  markViewed: []
  export: []
}>()

const canSearch = computed(() => !props.loading && !props.searchDisabled && Boolean(cnpj.value))

function onCompanyChange(value: string) {
  cnpj.value = value
  emit('companyChange')
}

function onOnlyUnviewedChange(value: boolean) {
  onlyUnviewed.value = value
  emit('search')
}

// onEnter searches when Enter is released in a text field of the bar. A
// q-select takes focus through a readonly input of its own, and Enter there
// belongs to the select, so it does not search.
function onEnter(event: KeyboardEvent) {
  const target = event.target
  if (!(target instanceof HTMLInputElement)) return
  if (target.readOnly || target.closest('.q-select')) return
  if (canSearch.value) emit('search')
}
</script>

<style>
/* Not scoped: the source's selects come through the default slot. The
   selector matches the specificity of Quasar's `.row > .col-md-auto`, which
   sets min-width: 0 and would otherwise win. */
.row > .document-filter-select {
  min-width: 130px;
}
</style>
