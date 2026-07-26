import { computed, ref, shallowRef } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import type { CompanySummary, CredentialSummary } from '@/types/desktop'

export function useCompanies() {
  const syncStore = useCompanySyncStore()
  const { syncing, syncingCNPJs } = storeToRefs(syncStore)
  const companies = shallowRef<CompanySummary[]>([])
  const credentials = shallowRef<CredentialSummary[]>([])

  const pendingLoads = ref(0)
  const loading = computed(() => pendingLoads.value > 0)

  async function trackLoad<T>(operation: () => Promise<T>): Promise<T> {
    pendingLoads.value++
    try {
      return await operation()
    } finally {
      pendingLoads.value--
    }
  }

  async function loadCredentials() {
    return trackLoad(async () => {
      credentials.value = await desktopClient.listCredentials()
      return credentials.value
    })
  }

  async function loadCompanies() {
    return trackLoad(async () => {
      companies.value = await desktopClient.listCompanies()
      return companies.value
    })
  }

  async function reloadData() {
    await Promise.all([loadCredentials(), loadCompanies()])
  }

  async function assignCredential(companyCNPJ: string, credentialID: string) {
    await desktopClient.assignCredential({
      CompanyCNPJ: companyCNPJ,
      CredentialID: credentialID,
    })
    await loadCompanies()
  }

  async function syncCompany(cnpj: string) {
    syncStore.startSync(cnpj)
    try {
      const result = await desktopClient.pull({ CNPJ: cnpj, Mode: '' })
      await loadCompanies()
      return result
    } finally {
      syncStore.finishSync(cnpj)
    }
  }

  async function resetSyncState(cnpj: string) {
    await desktopClient.resetSyncState({ CompanyCNPJ: cnpj })
    await loadCompanies()
  }

  return {
    companies,
    credentials,
    loading,
    syncing,
    syncingCNPJs,
    isSyncingCompany: syncStore.isSyncing,
    loadCompanies,
    loadCredentials,
    reloadData,
    assignCredential,
    syncCompany,
    resetSyncState,
  }
}
