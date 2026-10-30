import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { SyncSource } from '@/types/desktop'

function syncKey(cnpj: string, source: SyncSource) {
  return `${source}:${cnpj}`
}

function withoutKey<T>(record: Record<string, T>, key: string): Record<string, T> {
  return Object.fromEntries(Object.entries(record).filter(([other]) => other !== key))
}

// After a cancel, cancelSync asks every CANCEL_POLL_MS whether the backend
// still runs the sync, for at most CANCEL_WAIT_MS.
export const CANCEL_POLL_MS = 500
export const CANCEL_WAIT_MS = 15_000

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export const useCompanySyncStore = defineStore('companySync', () => {
  const activeSyncs = ref<Record<string, number>>({})
  // cancellers holds the AbortController of each cancellable sync in flight,
  // by the same key, so any screen can cancel it.
  const cancellers = shallowRef<Record<string, AbortController>>({})
  // cancelling holds, by the same key, the syncs cancelled in the frontend
  // that the backend may still be running. They count as syncing.
  const cancelling = ref<Record<string, true>>({})

  // anySyncing is true while a sync of any company or source runs.
  const anySyncing = computed(
    () => Object.keys(activeSyncs.value).length > 0 || Object.keys(cancelling.value).length > 0
  )

  function isSyncing(cnpj: string, source: SyncSource) {
    const key = syncKey(cnpj, source)
    return Boolean(activeSyncs.value[key] || cancelling.value[key])
  }

  function isCancelling(cnpj: string, source: SyncSource) {
    return Boolean(cancelling.value[syncKey(cnpj, source)])
  }

  function startSync(cnpj: string, source: SyncSource) {
    const key = syncKey(cnpj, source)
    activeSyncs.value = {
      ...activeSyncs.value,
      [key]: (activeSyncs.value[key] ?? 0) + 1,
    }
  }

  function finishSync(cnpj: string, source: SyncSource) {
    const key = syncKey(cnpj, source)
    const count = activeSyncs.value[key] ?? 0

    if (count <= 1) {
      activeSyncs.value = withoutKey(activeSyncs.value, key)
      return
    }

    activeSyncs.value = { ...activeSyncs.value, [key]: count - 1 }
  }

  // runSync marks the sync of the company's source as in flight while fn
  // runs, so every screen that shows it sees it busy.
  async function runSync<T>(cnpj: string, source: SyncSource, fn: () => Promise<T>): Promise<T> {
    startSync(cnpj, source)
    try {
      return await fn()
    } finally {
      finishSync(cnpj, source)
    }
  }

  // runCancellableSync is runSync for a call that stops when its signal
  // aborts. The controller is kept before the call starts and dropped when
  // the call settles, so cancelSync works from any screen while it runs.
  async function runCancellableSync<T>(
    cnpj: string,
    source: SyncSource,
    fn: (signal: AbortSignal) => Promise<T>
  ): Promise<T> {
    const key = syncKey(cnpj, source)
    const controller = new AbortController()
    cancellers.value = { ...cancellers.value, [key]: controller }
    try {
      return await runSync(cnpj, source, () => fn(controller.signal))
    } finally {
      if (cancellers.value[key] === controller) {
        cancellers.value = withoutKey(cancellers.value, key)
      }
    }
  }

  function canCancelSync(cnpj: string, source: SyncSource) {
    return Boolean(cancellers.value[syncKey(cnpj, source)])
  }

  // cancelSync cancels the company's cancellable sync of source. The call
  // rejects at once with the canceled code, but the backend keeps the source
  // reserved until it sees the cancel and records the run; a sync started
  // meanwhile would get sync_running. So the pair stays "cancelando", and
  // syncing, until stillRunning says the backend is done or CANCEL_WAIT_MS
  // pass. A failed stillRunning counts as still running.
  async function cancelSync(
    cnpj: string,
    source: SyncSource,
    stillRunning: () => Promise<boolean>
  ): Promise<void> {
    const key = syncKey(cnpj, source)
    const controller = cancellers.value[key]
    if (!controller || cancelling.value[key]) return
    cancelling.value = { ...cancelling.value, [key]: true }
    controller.abort()
    try {
      const deadline = Date.now() + CANCEL_WAIT_MS
      while (Date.now() < deadline) {
        await delay(CANCEL_POLL_MS)
        if (!(await stillRunning().catch(() => true))) return
      }
    } finally {
      cancelling.value = withoutKey(cancelling.value, key)
    }
  }

  return {
    activeSyncs,
    anySyncing,
    isSyncing,
    isCancelling,
    startSync,
    finishSync,
    runSync,
    runCancellableSync,
    canCancelSync,
    cancelSync,
  }
})
