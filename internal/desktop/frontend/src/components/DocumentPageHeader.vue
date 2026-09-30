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
        v-if="resetLabel"
        flat
        color="negative"
        icon="restart_alt"
        :label="resetLabel"
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
// ambiente badge, the sync and reset buttons, the status line and the banner
// that explains a blocked sync. Without resetLabel there is no reset button.
withDefaults(
  defineProps<{
    title: string
    ambiente?: { label: string; color: string } | null
    statusLine?: string
    blockedText?: string
    syncing: boolean
    syncDisabled: boolean
    resetLabel?: string
    resetTitle?: string
    resetting?: boolean
    resetDisabled?: boolean
  }>(),
  {
    ambiente: null,
    statusLine: '',
    blockedText: '',
    resetLabel: '',
    resetTitle: '',
    resetting: false,
    resetDisabled: false,
  }
)

const emit = defineEmits<{
  sync: []
  reset: []
}>()

const $q = useQuasar()
</script>
