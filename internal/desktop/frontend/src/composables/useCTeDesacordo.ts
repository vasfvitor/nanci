import { storeToRefs } from 'pinia'
import { useCTeLoaders } from '@/composables/useCTeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'

// useCTeDesacordo plans and sends the prestação do serviço em desacordo. The
// send can outlive the page, so the in-flight marker lives in the
// cteDocuments store and the list refresh does not depend on the page.
export function useCTeDesacordo() {
  const store = useCTeDocumentsStore()
  const { desacordoInFlight } = storeToRefs(store)
  const { refresh } = useCTeLoaders()

  // planDesacordo asks the backend what the event would send. It sends
  // nothing and asks for no password.
  function planDesacordo(chaveAcesso: string, observacao: string) {
    return desktopClient.planCTeDesacordo({
      CNPJ: store.listInput.CNPJ,
      ChaveAcesso: chaveAcesso,
      Observacao: observacao,
    })
  }

  // registerDesacordo sends the event. It returns null without a company or
  // while the same CT-e is being sent.
  async function registerDesacordo(chaveAcesso: string, observacao: string) {
    const cnpj = store.listInput.CNPJ
    if (!cnpj || desacordoInFlight.value.has(chaveAcesso)) return null

    desacordoInFlight.value.add(chaveAcesso)
    let result
    try {
      result = await desktopClient.registerCTeDesacordo({
        CNPJ: cnpj,
        ChaveAcesso: chaveAcesso,
        Observacao: observacao,
      })
    } finally {
      desacordoInFlight.value.delete(chaveAcesso)
    }

    await refresh(cnpj)
    return result
  }

  return {
    isSending: (chaveAcesso: string) => desacordoInFlight.value.has(chaveAcesso),
    planDesacordo,
    registerDesacordo,
  }
}
