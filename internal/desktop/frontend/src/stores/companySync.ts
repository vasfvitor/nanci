import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export const useCompanySyncStore = defineStore('companySync', () => {
  const activeSyncs = ref<Record<string, number>>({})

  const syncingCNPJs = computed(() => Object.keys(activeSyncs.value))
  const syncing = computed(() => syncingCNPJs.value[0] ?? null)

  function isSyncing(cnpj: string) {
    return Boolean(activeSyncs.value[cnpj])
  }

  function startSync(cnpj: string) {
    activeSyncs.value = {
      ...activeSyncs.value,
      [cnpj]: (activeSyncs.value[cnpj] ?? 0) + 1,
    }
  }

  function finishSync(cnpj: string) {
    const next = { ...activeSyncs.value }
    const count = next[cnpj] ?? 0

    if (count <= 1) {
      activeSyncs.value = Object.fromEntries(
        Object.entries(activeSyncs.value).filter(([activeCNPJ]) => activeCNPJ !== cnpj)
      )
      return
    } else {
      next[cnpj] = count - 1
    }

    activeSyncs.value = next
  }

  return {
    activeSyncs,
    syncingCNPJs,
    syncing,
    isSyncing,
    startSync,
    finishSync,
  }
})
