import { computed, ref, shallowRef } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CompanySummary, CredentialSummary } from '@/types/desktop'

export function useCompanies() {
  const syncStore = useCompanySyncStore()
  const workspace = useWorkspaceStore()
  // companies is the workspace list, so the drawer sees every change.
  const { companies } = storeToRefs(workspace)
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
    return trackLoad(() => workspace.loadCompanies())
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
    return syncStore.runSync(cnpj, 'nfse', async () => {
      const result = await desktopClient.pull({ CNPJ: cnpj, Mode: '' })
      await loadCompanies()
      return result
    })
  }

  async function resetSyncState(cnpj: string) {
    await desktopClient.resetSyncState({ CompanyCNPJ: cnpj })
    await loadCompanies()
  }

  return {
    companies,
    credentials,
    loading,
    isSyncingCompany: (cnpj: string) => syncStore.isSyncing(cnpj, 'nfse'),
    loadCompanies,
    loadCredentials,
    reloadData,
    assignCredential,
    syncCompany,
    resetSyncState,
  }
}

// companyOption is a company as a q-select option keyed by CNPJ.
export function companyOption(company: CompanySummary) {
  return {
    label: `${company.Name} (${company.CNPJ})`,
    value: company.CNPJ,
  }
}
