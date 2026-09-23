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

  function isAnySyncing(cnpj: string) {
    return isSyncing(cnpj, 'nfse') || isSyncing(cnpj, 'nfe')
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

  return {
    activeSyncs,
    isSyncing,
    isAnySyncing,
    startSync,
    finishSync,
  }
})
