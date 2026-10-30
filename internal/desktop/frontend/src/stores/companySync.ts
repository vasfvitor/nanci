import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import type { SyncSource } from '@/types/desktop'

function syncKey(cnpj: string, source: SyncSource) {
  return `${source}:${cnpj}`
}

function withoutKey<T>(record: Record<string, T>, key: string): Record<string, T> {
  return Object.fromEntries(Object.entries(record).filter(([other]) => other !== key))
}

export const useCompanySyncStore = defineStore('companySync', () => {
  const activeSyncs = ref<Record<string, number>>({})
  // cancellable holds, by the same key, the syncs in flight that the backend
  // can cancel (CancelSync), so any screen can cancel them.
  const cancellable = ref<Record<string, true>>({})
  // cancelling holds the cancellable syncs the user cancelled. The sync stays
  // in activeSyncs until its call settles, when the backend has stopped.
  const cancelling = ref<Record<string, true>>({})

  // anySyncing is true while a sync of any company or source runs.
  const anySyncing = computed(() => Object.keys(activeSyncs.value).length > 0)

  function isSyncing(cnpj: string, source: SyncSource) {
    return Boolean(activeSyncs.value[syncKey(cnpj, source)])
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

  // runCancellableSync is runSync for a Pull, PullNFe or PullCTe call, which
  // cancelSync can stop from any screen while it runs.
  async function runCancellableSync<T>(
    cnpj: string,
    source: SyncSource,
    fn: () => Promise<T>
  ): Promise<T> {
    const key = syncKey(cnpj, source)
    cancellable.value = { ...cancellable.value, [key]: true }
    try {
      return await runSync(cnpj, source, fn)
    } finally {
      cancellable.value = withoutKey(cancellable.value, key)
      cancelling.value = withoutKey(cancelling.value, key)
    }
  }

  function canCancelSync(cnpj: string, source: SyncSource) {
    const key = syncKey(cnpj, source)
    return Boolean(cancellable.value[key] && !cancelling.value[key])
  }

  // cancelSync asks the backend to stop the company's cancellable sync of
  // source. The sync call then rejects with the canceled code once the
  // backend has stopped and recorded the run; until then the pair shows
  // "cancelando" and stays syncing.
  async function cancelSync(cnpj: string, source: SyncSource): Promise<void> {
    const key = syncKey(cnpj, source)
    if (!canCancelSync(cnpj, source)) return
    cancelling.value = { ...cancelling.value, [key]: true }
    try {
      await desktopClient.cancelSync(cnpj, source)
    } catch (error) {
      cancelling.value = withoutKey(cancelling.value, key)
      throw error
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
