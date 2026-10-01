<template>
  <div class="row no-wrap items-center q-gutter-x-xs">
    <span
      :title="title"
      class="cursor-pointer text-weight-medium text-mono"
      @click="copyChave(chave)"
    >
      {{ formatChaveAcesso(chave) || '—' }}
    </span>
    <q-btn
      dense
      flat
      round
      size="xs"
      color="grey-7"
      icon="content_copy"
      title="Copiar chave completa"
      aria-label="Copiar chave completa"
      :disable="!chave"
      @click.stop="copyChave(chave)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { copyChave } from '@/composables/useNotify'
import { formatChaveAcesso, formatChaveDFe } from '@/utils/formatters'

// ChaveCell shows the end of an access key; the title has the whole key,
// grouped when it is an NF-e or CT-e key, and a click copies it.
const props = defineProps<{
  chave: string
}>()

const title = computed(() => formatChaveDFe(props.chave) || props.chave)
</script>
