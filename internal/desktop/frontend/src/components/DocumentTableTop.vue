<template>
  <div class="row items-center justify-between full-width">
    <div class="text-subtitle1 text-weight-bold">{{ title }}</div>
    <q-input
      v-model="text"
      class="document-search-input"
      placeholder="Filtrar por chave, número, nome ou CNPJ..."
      outlined
      dense
      clearable
      debounce="300"
    >
      <template #append>
        <q-icon name="search" />
      </template>
    </q-input>
  </div>
</template>

<script setup lang="ts">
// DocumentTableTop is the top of a document table: its title and the box
// that narrows the listed rows. The placeholder names only the fields that
// fit in the 350px box; each page searches a few more.
defineProps<{
  title: string
}>()

const text = defineModel<string>({ required: true })
</script>
