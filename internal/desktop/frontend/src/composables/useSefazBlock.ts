import { computed, getCurrentScope, onScopeDispose, shallowRef, watch, type Ref } from 'vue'
import type { ISODateValue } from '@/types/desktop'
import { formatTime, parseDate } from '@/utils/formatters'
import { blockedMessage, type SefazBlockInfo } from '@/utils/sefazDisplay'

export type SefazBlockStatus = SefazBlockInfo & { NextAllowedAt?: ISODateValue }

// useSefazBlock tells whether a SEFAZ distribution sync is paused, from the
// NextAllowedAt of the status, and why.
export function useSefazBlock(status: Readonly<Ref<SefazBlockStatus | null>>) {
  // now ticks when the block ends, so syncBlockedUntil clears itself without
  // polling.
  const now = shallowRef(Date.now())
  let unblockTimer: ReturnType<typeof setTimeout> | undefined

  watch(
    () => status.value?.NextAllowedAt,
    (value) => {
      now.value = Date.now()
      clearTimeout(unblockTimer)
      const until = parseDate(value)
      if (until && until.getTime() > now.value) {
        unblockTimer = setTimeout(() => {
          now.value = Date.now()
        }, until.getTime() - now.value + 1000)
      }
    },
    { immediate: true }
  )

  if (getCurrentScope()) {
    onScopeDispose(() => clearTimeout(unblockTimer))
  }

  const syncBlockedUntil = computed(() => {
    const until = parseDate(status.value?.NextAllowedAt)
    return until && until.getTime() > now.value ? until : null
  })

  const blockedText = computed(() => {
    if (!syncBlockedUntil.value || !status.value) return ''
    return blockedMessage(status.value, formatTime(syncBlockedUntil.value))
  })

  return { syncBlockedUntil, blockedText }
}
