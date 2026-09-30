<template>
  <div class="row no-wrap items-center">
    <q-btn
      dense
      flat
      round
      size="sm"
      :color="expanded ? 'primary' : 'grey-7'"
      :icon="expanded ? 'expand_less' : 'expand_more'"
      :title="`Ver detalhes ${noun}`"
      :aria-label="`Ver detalhes ${noun}`"
      :aria-expanded="expanded ? 'true' : 'false'"
      @click.stop="expanded = !expanded"
    />
    <q-btn dense flat round size="sm" color="grey-7" icon="more_vert" :aria-label="`Ações ${noun}`">
      <q-menu auto-close>
        <q-list dense class="row-actions-menu">
          <slot />
        </q-list>
      </q-menu>
    </q-btn>
  </div>
</template>

<script setup lang="ts">
// RowActionsMenu is the "Ações" cell of a document row: the toggle of the
// expanded details and a menu whose entries (RowMenuItem) go in the default
// slot. noun completes the labels: "da NFS-e", "da NF-e", "do CT-e".
defineProps<{
  noun: string
}>()

const expanded = defineModel<boolean>('expanded', { default: false })
</script>

<style scoped>
.row-actions-menu {
  min-width: 180px;
}
</style>
