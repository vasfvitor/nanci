<template>
  <q-dialog v-model="open">
    <q-card class="events-dialog">
      <q-card-section class="row items-center no-wrap q-pb-none">
        <div>
          <div class="text-h6">{{ title }}</div>
          <div class="text-caption text-app-muted text-mono">{{ formatChaveDFe(chaveAcesso) }}</div>
        </div>
        <q-space />
        <q-btn v-close-popup icon="close" flat round dense aria-label="Fechar" />
      </q-card-section>

      <q-card-section>
        <slot />
      </q-card-section>

      <q-card-actions align="right">
        <q-btn v-close-popup flat label="Fechar" color="primary" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { formatChaveDFe } from '@/utils/formatters'

// EventsDialogFrame is the shell of the events dialogs of the document pages:
// the title, the access key grouped as a subtitle, the events table in the
// default slot and a "Fechar" button.
defineProps<{
  title: string
  chaveAcesso: string
}>()

const open = defineModel<boolean>({ required: true })
</script>

<style scoped>
.events-dialog {
  min-width: 760px;
  max-width: 90vw;
}
</style>
