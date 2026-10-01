import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { SyncSource } from '@/types/desktop'

function syncKey(cnpj: string, source: SyncSource) {
  return `${source}:${cnpj}`
}

export const useCompanySyncStore = defineStore('companySync', () => {
  const activeSyncs = ref<Record<string, number>>({})

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

  return {
    activeSyncs,
    isSyncing,
    startSync,
    finishSync,
    runSync,
  }
})
