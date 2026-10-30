import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { SyncSource } from '@/types/desktop'

function syncKey(cnpj: string, source: SyncSource) {
  return `${source}:${cnpj}`
}

export const useCompanySyncStore = defineStore('companySync', () => {
  const activeSyncs = ref<Record<string, number>>({})
  // cancellers holds the AbortController of each cancellable sync in flight,
  // by the same key, so any screen can cancel it.
  const cancellers = shallowRef<Record<string, AbortController>>({})

  // anySyncing is true while a sync of any company or source runs.
  const anySyncing = computed(() => Object.keys(activeSyncs.value).length > 0)

  function isSyncing(cnpj: string, source: SyncSource) {
    return Boolean(activeSyncs.value[syncKey(cnpj, source)])
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
      activeSyncs.value = Object.fromEntries(
        Object.entries(activeSyncs.value).filter(([activeKey]) => activeKey !== key)
      )
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
        cancellers.value = Object.fromEntries(
          Object.entries(cancellers.value).filter(([activeKey]) => activeKey !== key)
        )
      }
    }
  }

  function canCancelSync(cnpj: string, source: SyncSource) {
    return Boolean(cancellers.value[syncKey(cnpj, source)])
  }

  // cancelSync cancels the company's cancellable sync of source. The call
  // rejects at once with the canceled code; the backend stops at its next
  // check of the context.
  function cancelSync(cnpj: string, source: SyncSource) {
    cancellers.value[syncKey(cnpj, source)]?.abort()
  }

  return {
    activeSyncs,
    anySyncing,
    isSyncing,
    startSync,
    finishSync,
    runSync,
    runCancellableSync,
    canCancelSync,
    cancelSync,
  }
})
