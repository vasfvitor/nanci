import { computed, ref, shallowRef } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CompanySummary, CredentialSummary } from '@/types/desktop'
import { formatCpfCnpj } from '@/utils/formatters'

export function useCompanies() {
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

  // loadCompanies lists the companies again, as after a change.
  async function loadCompanies() {
    return trackLoad(() => workspace.loadCompanies())
  }

  async function assignCredential(companyCNPJ: string, credentialID: string) {
    await desktopClient.assignCredential({
      CompanyCNPJ: companyCNPJ,
      CredentialID: credentialID,
    })
    await loadCompanies()
  }

  return {
    companies,
    credentials,
    loading,
    loadCompanies,
    loadCredentials,
    assignCredential,
  }
}

// companyOption is a company as a q-select option keyed by CNPJ, labeled
// "Name (12.345.678/0001-00)".
export function companyOption(company: CompanySummary) {
  return {
    label: `${company.Name} (${formatCpfCnpj(company.CNPJ)})`,
    value: company.CNPJ,
  }
}
