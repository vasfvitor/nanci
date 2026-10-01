<template>
  <div class="row no-wrap items-center">
    <q-btn
      dense
      flat
      round
      size="sm"
      :color="expanded ? 'primary' : 'grey-7'"
      :icon="expanded ? 'expand_less' : 'expand_more'"
      :title="`Ver detalhes ${ofNoun}`"
      :aria-label="`Ver detalhes ${ofNoun}`"
      :aria-expanded="expanded ? 'true' : 'false'"
      @click.stop="expanded = !expanded"
    />
    <q-btn dense flat round size="sm" color="grey-7" icon="more_vert" :aria-label="`Ações ${ofNoun}`">
      <q-menu auto-close>
        <q-list dense class="row-actions-menu">
          <slot />
        </q-list>
      </q-menu>
    </q-btn>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { DOCUMENT_SOURCES, type DocumentSource } from '@/utils/documentSources'

// RowActionsMenu is the "Ações" cell of a document row: the toggle of the
// expanded details and a menu whose entries (RowMenuItem) go in the default
// slot. The labels name the source: "Ver detalhes do CT-e".
const props = defineProps<{
  source: DocumentSource
}>()

const ofNoun = computed(() => {
  const { article, noun } = DOCUMENT_SOURCES[props.source]
  return `${article} ${noun}`
})

const expanded = defineModel<boolean>('expanded', { default: false })
</script>

<style scoped>
.row-actions-menu {
  min-width: 180px;
}
</style>
