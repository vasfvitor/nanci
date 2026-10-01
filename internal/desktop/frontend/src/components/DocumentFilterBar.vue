<template>
  <div class="row q-gutter-sm items-center q-mb-md q-pa-sm rounded-borders shadow-1">
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

    <div class="col-12 col-sm-6 col-md-3" title="Competência pelo mês de emissão">
      <CompetencePicker v-model="competence" :disable="loading" @keyup.enter="search" />
    </div>

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
import CompetencePicker from './CompetencePicker.vue'

// DocumentFilterBar is the filter row of a document page, in the standard
// order: company, competência, the source's own filters (default slot),
// "Somente não vistos", then Buscar, the source's actions (#actions),
// "Marcar vistos (n)" and Exportar. Changing the company or the toggle
// searches, via companyChange and search. Enter in the competência searches;
// the default slot gets search to bind on its own text fields.
const props = withDefaults(
  defineProps<{
    companyOptions: { label: string; value: string }[]
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

const cnpj = defineModel<string>('cnpj', { required: true })
const competence = defineModel<string>('competence', { required: true })
const onlyUnviewed = defineModel<boolean>('onlyUnviewed', { required: true })

const emit = defineEmits<{
  search: []
  companyChange: []
  markViewed: []
  export: []
}>()

const canSearch = computed(() => !props.loading && !props.searchDisabled && Boolean(cnpj.value))

// search asks for a search when one can be sent now.
function search() {
  if (canSearch.value) emit('search')
}

function onCompanyChange(value: string) {
  cnpj.value = value
  emit('companyChange')
}

function onOnlyUnviewedChange(value: boolean) {
  onlyUnviewed.value = value
  emit('search')
}
</script>

