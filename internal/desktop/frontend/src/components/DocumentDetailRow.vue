<template>
  <q-tr :props="rowProps" :class="$q.dark.isActive ? 'bg-grey-10' : 'bg-grey-1'">
    <q-td :colspan="colspan ?? rowProps.cols.length + 1" class="document-detail-cell">
      <div class="document-detail q-pa-md">
        <slot />
      </div>
    </q-td>
  </q-tr>
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'

// DocumentDetailRow is the expanded details of a document row; the page
// renders it under v-if="rowProps.expand". rowProps is the q-table #body
// slot scope. The default colspan covers every column plus the selection one.
defineProps<{
  rowProps: { cols: readonly unknown[] }
  colspan?: number
}>()

const $q = useQuasar()
</script>

<style scoped>
/* The details span every column, which is wider than the visible table when
   it scrolls sideways. They are sized to the scroll area (100cqw, see
   .document-table in app.scss) and pinned to its left edge, so they wrap
   inside what the user sees. */
.document-detail {
  position: sticky;
  left: 0;
  width: 100cqw;
  overflow-wrap: anywhere;
}
</style>
