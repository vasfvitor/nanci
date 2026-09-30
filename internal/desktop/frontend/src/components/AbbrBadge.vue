<template>
  <q-badge
    v-bind="badgeProps(color, $q.dark.isActive)"
    :label="abbr"
    :aria-label="description"
    class="abbr-badge text-weight-bold text-mono cursor-help"
    :class="{ 'abbr-badge--secondary': secondary }"
  >
    <q-tooltip>{{ description }}</q-tooltip>
  </q-badge>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useQuasar } from 'quasar'
import { badgeProps } from '@/utils/sefazDisplay'

// AbbrBadge is a state badge that shows an abbreviation and explains it in
// a tooltip and to screen readers as "Kind: Label".
const props = defineProps<{
  abbr: string
  label: string
  color: string
  kind?: string
  secondary?: boolean
}>()

const $q = useQuasar()

const description = computed(() => (props.kind ? `${props.kind}: ${props.label}` : props.label))
</script>

<style scoped>
.abbr-badge--secondary {
  font-size: 10px;
}
</style>
