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
        v-if="syncing && (cancelable || cancelling)"
        flat
        color="primary"
        icon="close"
        :label="cancelling ? 'Cancelando…' : 'Cancelar'"
        :disable="cancelling"
        title="Interrompe a sincronização; o que já foi salvo fica, e a próxima continua dali"
        @click="emit('cancel')"
      />
      <q-btn
        color="primary"
        icon="sync"
        :label="`Sincronizar ${title}`"
        :loading="syncing"
        :disable="syncDisabled"
        :class="{ 'first-sync': firstSync }"
        @click="emit('sync')"
      >
        <template v-if="firstSync">
          <q-badge floating color="accent" label="Primeira sincronização" />
          <q-tooltip>Busque as notas da empresa pela primeira vez.</q-tooltip>
        </template>
      </q-btn>
    </div>

    <!-- The workspace is picked in the left drawer, which Quasar hides below
         the md breakpoint, so the page always names it here. -->
    <div v-if="contextLine" class="text-body2 q-mb-xs">{{ contextLine }}</div>

    <div v-if="statusLine" class="text-caption text-app-muted q-mb-sm">{{ statusLine }}</div>

    <div v-if="idleWarning" class="q-mb-sm">
      <q-badge
        v-bind="badgeProps('warning', $q.dark.isActive)"
        :label="idleWarning"
        class="idle-warning cursor-help"
      >
        <q-tooltip max-width="22rem">{{ IDLE_WARNING_TOOLTIP }}</q-tooltip>
      </q-badge>
    </div>

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
import { badgeProps, IDLE_WARNING_TOOLTIP } from '@/utils/sefazDisplay'

// DocumentPageHeader is the top of a document page: the title with the
// ambiente badge, the sync and reset buttons, the workspace company and
// competência, the status line, the badge that warns of a distribution idle
// for too long and the banner that explains a blocked sync. ambiente is null
// until the company is known; contextLine is '' while no company is
// selected; resetTitle explains what the reset does; firstSync marks the
// sync button while the company was never synced; cancelable shows the
// Cancelar button next to a running sync, and cancelling turns it into a
// disabled "Cancelando…" until the backend has stopped.
withDefaults(
  defineProps<{
    title: string
    ambiente: { label: string; color: string } | null
    contextLine: string
    statusLine?: string
    idleWarning?: string
    blockedText?: string
    syncing: boolean
    syncDisabled: boolean
    resetTitle: string
    resetting: boolean
    resetDisabled: boolean
    firstSync?: boolean
    cancelable?: boolean
    cancelling?: boolean
  }>(),
  {
    statusLine: '',
    idleWarning: '',
    blockedText: '',
    firstSync: false,
    cancelable: false,
    cancelling: false,
  }
)

const emit = defineEmits<{
  sync: []
  reset: []
  cancel: []
}>()

const $q = useQuasar()
</script>

<style scoped>
/* currentColor is the white label of the primary button, which would vanish
   on a light page; the outline takes the primary colour instead. */
.first-sync {
  animation: first-sync-pulse 1.6s ease-in-out infinite;
}

@keyframes first-sync-pulse {
  0%,
  100% {
    outline: 2px solid transparent;
    outline-offset: 0;
  }
  50% {
    outline: 2px solid var(--q-primary);
    outline-offset: 3px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .first-sync {
    animation: none;
    outline: 2px solid var(--q-primary);
    outline-offset: 3px;
  }
}
</style>
