<template>
  <q-expansion-item
    v-model="open"
    dense
    dense-toggle
    expand-separator
    icon="help_outline"
    label="Legenda dos estados"
    header-class="text-weight-medium"
    class="state-legend rounded-borders"
    :class="$q.dark.isActive ? 'bg-grey-10' : 'bg-grey-1'"
  >
    <div v-if="open" class="q-px-md q-pb-md row q-col-gutter-md">
      <section
        v-for="section in sections"
        :key="section.title"
        class="col-12 col-md-6 col-lg-4 state-legend-section"
        :data-legend-section="section.title"
      >
        <div class="text-subtitle2 q-mb-xs">{{ section.title }}</div>
        <div v-if="section.note" class="text-caption text-app-muted q-mb-xs">{{ section.note }}</div>
        <q-list dense>
          <q-item v-for="entry in section.items" :key="entry.badge" dense class="q-px-none state-legend-item">
            <q-item-section side class="state-legend-badge">
              <q-chip
                v-if="entry.outline"
                dense
                square
                outline
                size="sm"
                :color="entry.color"
                :label="entry.badge"
                class="q-ma-none"
              />
              <q-badge
                v-else
                v-bind="badgeProps(entry.color, $q.dark.isActive)"
                :label="entry.badge"
                class="text-weight-bold"
                :class="{ 'text-mono': Boolean(entry.name) }"
              />
            </q-item-section>
            <q-item-section>
              <q-item-label>
                <span v-if="entry.name" class="text-weight-medium">{{ entry.name }}: </span>
                <span>{{ entry.description }}</span>
              </q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </section>
    </div>
  </q-expansion-item>
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'
import { badgeProps } from '@/utils/sefazDisplay'
import type { LegendSection } from '@/utils/stateLegends'

// StateLegend explains the state badges of a document table. It starts
// collapsed and renders the sections only while open.
defineProps<{
  sections: LegendSection[]
}>()

const open = defineModel<boolean>({ default: false })

const $q = useQuasar()
</script>

<style scoped>
.state-legend-badge {
  min-width: 5.5rem;
  align-items: flex-start;
}
</style>
