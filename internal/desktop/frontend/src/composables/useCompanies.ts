import { ref, shallowRef } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary, CredentialSummary } from '@/types/desktop'

export function useCompanies() {
  const companies = shallowRef<CompanySummary[]>([])
  const credentials = shallowRef<CredentialSummary[]>([])
  const loading = ref(false)
  const syncing = ref<string | null>(null)

  async function loadCredentials() {
    credentials.value = await desktopClient.listCredentials()
    return credentials.value
  }

  async function loadCompanies() {
    companies.value = await desktopClient.listCompanies()
    return companies.value
  }

  async function reloadData() {
    loading.value = true
    try {
      await Promise.all([loadCredentials(), loadCompanies()])
    } finally {
      loading.value = false
    }
  }

  async function assignCredential(companyCNPJ: string, credentialID: string) {
    await desktopClient.assignCredential({
      CompanyCNPJ: companyCNPJ,
      CredentialID: credentialID,
    })
    await loadCompanies()
  }

  async function syncCompany(cnpj: string) {
    syncing.value = cnpj
    try {
      const result = await desktopClient.pull({ CNPJ: cnpj, Mode: '' })
      await loadCompanies()
      return result
    } finally {
      syncing.value = null
    }
  }

  async function resetSyncState(cnpj: string) {
    await desktopClient.resetSyncState({ CNPJ: cnpj })
    await loadCompanies()
  }

  return {
    companies,
    credentials,
    loading,
    syncing,
    loadCompanies,
    loadCredentials,
    reloadData,
    assignCredential,
    syncCompany,
    resetSyncState,
  }
}
