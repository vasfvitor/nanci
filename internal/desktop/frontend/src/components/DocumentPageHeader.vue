<template>
  <div>
    <div class="row items-center q-gutter-sm q-mb-sm">
      <h5 class="q-my-none">{{ title }}</h5>
      <q-badge
        v-if="ambiente"
        v-bind="badgeProps(ambiente.color, $q.dark.isActive)"
        :label="ambiente.label"
        class="text-weight-bold"
      />
      <q-space />
      <q-btn
        flat
        color="negative"
        icon="restart_alt"
        :label="`Redefinir ${title}`"
        :title="resetTitle"
        :loading="resetting"
        :disable="resetDisabled"
        @click="emit('reset')"
      />
      <q-btn
        color="primary"
        icon="sync"
        :label="`Sincronizar ${title}`"
        :loading="syncing"
        :disable="syncDisabled"
        @click="emit('sync')"
      />
    </div>

    <!-- The workspace is picked in the left drawer, which Quasar hides below
         the md breakpoint, so the page always names it here. -->
    <div v-if="contextLine" class="text-body2 q-mb-xs">{{ contextLine }}</div>

    <div v-if="statusLine" class="text-caption text-app-muted q-mb-sm">{{ statusLine }}</div>

    <q-banner
      v-if="blockedText"
      dense
      rounded
      class="q-mb-md"
      :class="$q.dark.isActive ? 'bg-grey-9 text-orange-3' : 'bg-orange-1 text-orange-10'"
    >
      <template #avatar>
        <q-icon name="schedule" />
      </template>
      {{ blockedText }}
    </q-banner>
  </div>
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'
import { badgeProps } from '@/utils/sefazDisplay'

// DocumentPageHeader is the top of a document page: the title with the
// ambiente badge, the sync and reset buttons, the workspace company and
// competência, the status line and the banner that explains a blocked sync.
// ambiente is null until the company is known; contextLine is '' while no
// company is selected; resetTitle explains what the reset does.
withDefaults(
  defineProps<{
    title: string
    ambiente: { label: string; color: string } | null
    contextLine: string
    statusLine?: string
    blockedText?: string
    syncing: boolean
    syncDisabled: boolean
    resetTitle: string
    resetting: boolean
    resetDisabled: boolean
  }>(),
  {
    statusLine: '',
    blockedText: '',
  }
)

const emit = defineEmits<{
  sync: []
  reset: []
}>()

const $q = useQuasar()
</script>
