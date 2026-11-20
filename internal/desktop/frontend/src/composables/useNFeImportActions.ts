import type { Ref } from 'vue'
import { useQuasar } from 'quasar'
import NFeFetchDialog from '@/components/NFeFetchDialog.vue'
import { useNotify } from '@/composables/useNotify'
import type { FetchByChaveResult, NFeStatusResult } from '@/types/desktop'
import { asSentence } from '@/utils/formatters'

export type NFeImportActionsOptions = {
  cnpj: Ref<string>
  // status gives the hourly request budget the fetch dialog shows.
  status: Ref<NFeStatusResult | null>
  // fetchByChave returns null when it did not run.
  fetchByChave: (chaveAcesso: string) => Promise<FetchByChaveResult | null>
}

// useNFeImportActions holds the "Buscar pela chave" button of the NF-e
// page: the fetch dialog and the notifications. The request goes through
// useNFeDocuments, which keeps its marker. "Importar XML" is
// useImportActions.
export function useNFeImportActions(options: NFeImportActionsOptions) {
  const $q = useQuasar()
  const { notifySuccess, notifyInfo, notifyWarning, notifySyncError } = useNotify()

  function openFetchDialog() {
    if (!options.cnpj.value) return
    $q.dialog({
      component: NFeFetchDialog,
      componentProps: {
        requestsLastHour: options.status.value?.RequestsLastHour ?? 0,
        requestBudget: options.status.value?.RequestBudget ?? 0,
      },
    }).onOk((chave: string) => {
      void fetchByChave(chave)
    })
  }

  async function fetchByChave(chave: string) {
    try {
      const result = await options.fetchByChave(chave)
      if (result) notifyFetchResult(result)
    } catch (error) {
      notifySyncError('Erro ao buscar a NF-e', error)
    }
  }

  function notifyFetchResult(result: FetchByChaveResult) {
    switch (result.Outcome) {
      case 'completa':
        notifySuccess('NF-e completa salva.')
        return
      case 'resumo':
        notifyInfo(asSentence(result.Message) || 'A SEFAZ entregou só o resumo da NF-e.')
        return
      default:
        notifyWarning(asSentence(result.Message) || 'A SEFAZ não encontrou a NF-e.')
    }
  }

  return { openFetchDialog }
}
