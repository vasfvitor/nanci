import { storeToRefs } from 'pinia'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeConclusiveTipo } from '@/types/desktop'

// useNFeManifestation sends manifestação do destinatário events. The calls
// can outlive the page, so in-flight markers and results live in the
// nfeDocuments store and the follow-up refresh does not depend on the page.
export function useNFeManifestation() {
  const store = useNFeDocumentsStore()
  const { selected, pending, pendingLoading, cienciaInFlight, manifestationInFlight } =
    storeToRefs(store)
  const { loadPending, refresh } = useNFeLoaders()

  // planCiencia asks the backend which notes a ciência would send. It sends
  // nothing and asks for no password.
  async function planCiencia(chavesAcesso: string[]) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || chavesAcesso.length === 0) return null
    return desktopClient.planCiencia(cnpj, chavesAcesso)
  }

  async function registerCiencia(chavesAcesso: string[]) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || chavesAcesso.length === 0) return null
    if (cienciaInFlight.value || chavesAcesso.some((chave) => store.isChaveBusy(chave))) return null

    cienciaInFlight.value = [...chavesAcesso]
    let result
    try {
      result = await desktopClient.registerCiencia(cnpj, chavesAcesso)
    } finally {
      cienciaInFlight.value = null
    }

    selected.value = []
    await refresh(cnpj)
    return result
  }

  async function registerManifestation(
    chaveAcesso: string,
    tipo: NFeConclusiveTipo,
    justificativa = ''
  ) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || store.isChaveBusy(chaveAcesso)) return null

    manifestationInFlight.value.add(chaveAcesso)
    let result
    try {
      result = await desktopClient.registerManifestation({
        CNPJ: cnpj,
        ChaveAcesso: chaveAcesso,
        Tipo: tipo,
        Justificativa: tipo === '210240' ? justificativa.trim() : '',
      })
    } finally {
      manifestationInFlight.value.delete(chaveAcesso)
    }

    await refresh(cnpj)
    return result
  }

  return {
    pending,
    pendingLoading,
    cienciaInFlight,
    manifestationInFlight,
    isChaveBusy: (chaveAcesso: string) => store.isChaveBusy(chaveAcesso),
    loadPending,
    planCiencia,
    registerCiencia,
    registerManifestation,
  }
}
