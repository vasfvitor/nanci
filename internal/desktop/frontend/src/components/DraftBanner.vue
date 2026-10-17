<template>
  <q-banner
    dense
    rounded
    :class="$q.dark.isActive ? 'bg-grey-9 text-grey-3' : 'bg-blue-1 text-blue-10'"
  >
    <template #avatar>
      <q-icon name="history" />
    </template>
    Rascunho recuperado: os dados digitados antes de fechar foram mantidos.
    <template #action>
      <q-btn flat dense label="Limpar" @click="emit('clear')" />
    </template>
  </q-banner>
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'

const emit = defineEmits<{
  clear: []
}>()
const $q = useQuasar()
</script>
