import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeConclusiveTipo } from '@/types/desktop'

// useNFeManifestation sends manifestação do destinatário events. The calls
// can outlive the page, so in-flight markers and results live in the
// nfeDocuments store and the follow-up refresh does not depend on the page.
export function useNFeManifestation() {
  const store = useNFeDocumentsStore()
  const { pending, pendingLoading, cienciaInFlight, manifestationInFlight, lastCienciaResult } =
    storeToRefs(store)

  async function loadPending(cnpj: string = store.filter.CNPJ) {
    if (!cnpj) {
      store.setPending([])
      return []
    }
    pendingLoading.value = true
    try {
      const rows = await desktopClient.listPendingManifestations(cnpj)
      if (store.filter.CNPJ === cnpj) {
        store.setPending(rows)
      }
      return rows
    } finally {
      pendingLoading.value = false
    }
  }

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
    if (store.cienciaInFlight || chavesAcesso.some((chave) => store.isChaveBusy(chave))) return null

    store.startCiencia(cnpj, chavesAcesso)
    let result
    try {
      result = await desktopClient.registerCiencia(cnpj, chavesAcesso)
    } finally {
      store.finishCiencia()
    }

    store.setLastCienciaResult(result)
    store.clearSelection()
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

    store.startManifestation(chaveAcesso, tipo)
    let result
    try {
      result = await desktopClient.registerManifestation({
        CNPJ: cnpj,
        ChaveAcesso: chaveAcesso,
        Tipo: tipo,
        Justificativa: tipo === '210240' ? justificativa.trim() : '',
      })
    } finally {
      store.finishManifestation(chaveAcesso)
    }

    await refresh(cnpj)
    return result
  }

  // refresh reloads notes, pendências and status after an event was sent.
  // The event is already registered at SEFAZ, so a failed refresh must not
  // turn the call into an error; the next search shows the current state.
  async function refresh(cnpj: string) {
    if (store.filter.CNPJ !== cnpj) return
    const input = store.listInput
    await Promise.allSettled([
      desktopClient.listNFe(input).then((rows) => {
        if (store.filter.CNPJ === cnpj) store.setRows(rows)
      }),
      desktopClient.statusNFe(cnpj).then((status) => {
        if (store.filter.CNPJ === cnpj) store.setStatus(status)
      }),
      loadPending(cnpj),
    ])
  }

  return {
    pending,
    pendingLoading,
    cienciaInFlight,
    manifestationInFlight,
    lastCienciaResult,
    isChaveBusy: (chaveAcesso: string) => store.isChaveBusy(chaveAcesso),
    loadPending,
    planCiencia,
    registerCiencia,
    registerManifestation,
  }
}
