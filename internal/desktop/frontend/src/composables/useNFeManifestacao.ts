import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeConclusiveTipo } from '@/types/desktop'
import { cienciaBlockReason } from '@/utils/nfeManifestacao'

// useNFeManifestacao sends manifestação do destinatário events. The calls
// can outlive the page, so in-flight markers and results live in the
// nfeDocuments store and the follow-up refresh does not depend on the page.
export function useNFeManifestacao() {
  const store = useNFeDocumentsStore()
  const { selected, pending, pendingLoading, planningCiencia, cienciaInFlight, manifestacaoInFlight } =
    storeToRefs(store)
  const { loadPending, refresh, refreshNote } = useNFeLoaders()

  // eligibleSelection is the selected notes that can receive ciência.
  const eligibleSelection = computed(() => selected.value.filter((row) => !cienciaBlockReason(row)))

  // planCiencia asks the backend which notes a ciência would send. It sends
  // nothing and asks for no password. It returns null while another plan is
  // in flight.
  async function planCiencia(chavesAcesso: string[]) {
    const cnpj = store.filter.CNPJ
    if (!cnpj || chavesAcesso.length === 0 || planningCiencia.value) return null

    planningCiencia.value = true
    try {
      return await desktopClient.planNFeCiencia(cnpj, chavesAcesso)
    } finally {
      planningCiencia.value = false
    }
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
    planningCiencia,
    cienciaInFlight,
    manifestacaoInFlight,
    eligibleSelection,
    isChaveBusy: (chaveAcesso: string) => store.isChaveBusy(chaveAcesso),
    loadPending,
    planCiencia,
    registerCiencia,
    registerManifestacao,
  }
}
