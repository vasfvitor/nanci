<template>
  <StateBadges :badges="badges">
    <template v-if="busy || showsDeadline" #default>
      <q-spinner v-if="busy" size="xs" color="primary" />
      <q-chip
        v-if="showsDeadline"
        dense
        square
        outline
        size="sm"
        class="q-ma-none"
        :color="deadlineColor(row.DaysLeft, 'conclusiva')"
        :label="conclusiveDeadlineLabel(row.DaysLeft)"
        :title="`Prazo da manifestação conclusiva: ${formatDate(row.ConclusiveDue)}`"
      />
    </template>
  </StateBadges>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StateBadges from './StateBadges.vue'
import type { NFeRow } from '@/types/desktop'
import { formatDate } from '@/utils/formatters'
import { conclusiveDeadlineLabel, deadlineColor, showsConclusiveDeadline } from '@/utils/nfeDisplay'
import type { StateBadge } from '@/utils/sefazDisplay'

// NFeStateBadges is the "Estados" cell of an NF-e row: the row's badges,
// a spinner while an event of the note is being sent and, after the
// ciência, the deadline of the conclusive manifestação.
const props = defineProps<{
  row: NFeRow
  badges: StateBadge[]
  busy: boolean
}>()

const showsDeadline = computed(() => showsConclusiveDeadline(props.row))
</script>
