import { storeToRefs } from 'pinia'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeConclusiveTipo } from '@/types/desktop'

// useNFeManifestacao sends manifestação do destinatário events. The calls
// can outlive the page, so in-flight markers and results live in the
// nfeDocuments store and the follow-up refresh does not depend on the page.
export function useNFeManifestacao() {
  const store = useNFeDocumentsStore()
  const { selected, pending, pendingLoading, cienciaInFlight, manifestacaoInFlight } =
    storeToRefs(store)
  const { loadPending, refresh, refreshNote } = useNFeLoaders()

  // planCiencia asks the backend which notes a ciência would send. It sends
  // nothing and asks for no password.
  async function planCiencia(chavesAcesso: string[]) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || chavesAcesso.length === 0) return null
    return desktopClient.planNFeCiencia(cnpj, chavesAcesso)
  }

  async function registerCiencia(chavesAcesso: string[]) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || chavesAcesso.length === 0) return null
    if (cienciaInFlight.value || chavesAcesso.some((chave) => store.isChaveBusy(chave))) return null

    cienciaInFlight.value = [...chavesAcesso]
    let result
    try {
      result = await desktopClient.registerNFeCiencia(cnpj, chavesAcesso)
    } finally {
      cienciaInFlight.value = null
    }

    selected.value = []
    await refresh(cnpj)
    return result
  }

  async function registerManifestacao(
    chaveAcesso: string,
    tipo: NFeConclusiveTipo,
    justificativa = ''
  ) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || store.isChaveBusy(chaveAcesso)) return null

    manifestacaoInFlight.value.add(chaveAcesso)
    let result
    try {
      result = await desktopClient.registerNFeManifestacao({
        CNPJ: cnpj,
        ChaveAcesso: chaveAcesso,
        Tipo: tipo,
        Justificativa: tipo === '210240' ? justificativa.trim() : '',
      })
    } finally {
      manifestacaoInFlight.value.delete(chaveAcesso)
    }

    await refreshNote(cnpj, chaveAcesso)
    return result
  }

  return {
    pending,
    pendingLoading,
    cienciaInFlight,
    manifestacaoInFlight,
    isChaveBusy: (chaveAcesso: string) => store.isChaveBusy(chaveAcesso),
    loadPending,
    planCiencia,
    registerCiencia,
    registerManifestacao,
  }
}
