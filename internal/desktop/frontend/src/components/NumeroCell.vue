<template>
  <div :title="title">
    <div class="text-mono ellipsis numero-cell">{{ numero || '—' }}</div>
    <div v-if="serie || !viewedAt" class="row no-wrap items-center q-gutter-x-xs">
      <span v-if="serie" class="text-caption text-app-muted">série {{ serie }}</span>
      <q-badge
        v-if="!viewedAt"
        v-bind="badgeProps(VIEWED_BADGE.color, $q.dark.isActive)"
        :label="VIEWED_BADGE.label"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useQuasar } from 'quasar'
import type { ISODateValue } from '@/types/desktop'
import { badgeProps, VIEWED_BADGE } from '@/utils/sefazDisplay'

// NumeroCell is the "Nº / Série" cell of a document table: the number, the
// série under it when the source has one, and the "Novo" badge while the
// document was not viewed. The title has the number and série in full.
const props = defineProps<{
  numero: string
  serie?: string
  viewedAt: ISODateValue
}>()

const $q = useQuasar()

const title = computed(() => [props.numero, props.serie].filter(Boolean).join(' / '))
</script>
