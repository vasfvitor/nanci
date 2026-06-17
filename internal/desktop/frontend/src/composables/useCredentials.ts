import { ref } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { CredentialSummary } from '@/types/desktop'

export function useCredentials() {
  const credentials = ref<CredentialSummary[]>([])
  const loading = ref(false)

  async function loadCredentials() {
    loading.value = true
    try {
      credentials.value = await desktopClient.listCredentials()
      return credentials.value
    } finally {
      loading.value = false
    }
  }

  async function selectCertificate() {
    return desktopClient.selectCertificate()
  }

  async function updateCredentialPath(credentialID: string, certPath: string) {
    await desktopClient.updateCredentialPath({ CredentialID: credentialID, CertPath: certPath })
    await loadCredentials()
  }

  async function updateCredentialData(credentialID: string, label: string) {
    await desktopClient.updateCredentialData({ CredentialID: credentialID, Label: label })
    await loadCredentials()
  }

  return {
    credentials,
    loading,
    loadCredentials,
    selectCertificate,
    updateCredentialPath,
    updateCredentialData,
  }
}
