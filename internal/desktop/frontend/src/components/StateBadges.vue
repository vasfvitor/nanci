<template>
  <div class="column items-center no-wrap">
    <div class="row no-wrap q-gutter-x-xs">
      <AbbrBadge
        v-for="badge in primary"
        :key="badge.key"
        :abbr="badge.abbr"
        :label="badge.label"
        :color="badge.color"
        :kind="badge.kind"
      />
    </div>
    <div
      v-if="secondary.length > 0 || slots.default"
      class="row no-wrap items-center q-gutter-x-xs state-badges-secondary"
    >
      <AbbrBadge
        v-for="badge in secondary"
        :key="badge.key"
        :abbr="badge.abbr"
        :label="badge.label"
        :color="badge.color"
        :kind="badge.kind"
        secondary
      />
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AbbrBadge from './AbbrBadge.vue'
import type { StateBadge } from '@/utils/sefazDisplay'

// StateBadges is the content of the "Estados" column: the row's badges on one
// line and, under them, the secondary badges followed by the default slot
// (a deadline chip, a spinner).
const props = defineProps<{
  badges: StateBadge[]
}>()

const slots = defineSlots<{ default?: () => unknown }>()

const primary = computed(() => props.badges.filter((badge) => !badge.secondary))
const secondary = computed(() => props.badges.filter((badge) => badge.secondary))
</script>

<style scoped>
.state-badges-secondary {
  margin-top: 2px;
}
</style>
