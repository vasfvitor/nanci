import { ref } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { AddCredentialInput, CredentialSummary } from '@/types/desktop'

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

  // addCredential and updateCredentialData only save: the page lists the
  // credentials again when the form dialog says it saved.
  function addCredential(input: AddCredentialInput) {
    return desktopClient.addCredential(input)
  }

  function updateCredentialData(credentialID: string, label: string) {
    return desktopClient.updateCredentialData({ CredentialID: credentialID, Label: label })
  }

  return {
    credentials,
    loading,
    loadCredentials,
    selectCertificate,
    updateCredentialPath,
    addCredential,
    updateCredentialData,
  }
}
