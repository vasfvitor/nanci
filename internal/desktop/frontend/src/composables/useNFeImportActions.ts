import type { Ref } from 'vue'
import { useQuasar } from 'quasar'
import ImportResultDialog from '@/components/ImportResultDialog.vue'
import NFeFetchDialog from '@/components/NFeFetchDialog.vue'
import { useNotify } from '@/composables/useNotify'
import type { FetchByChaveResult, ImportXMLResult, NFeStatusResult } from '@/types/desktop'
import { importSummary } from '@/utils/importDisplay'

export type NFeImportActionsOptions = {
  cnpj: Ref<string>
  // status gives the hourly request budget the fetch dialog shows.
  status: Ref<NFeStatusResult | null>
  // importXML and fetchByChave return null when they did not run.
  importXML: () => Promise<ImportXMLResult | null>
  fetchByChave: (chaveAcesso: string) => Promise<FetchByChaveResult | null>
}

// The backend message is a lowercase clause ("a SEFAZ entregou só o
// resumo: ..."); the notice shows it as a sentence.
function sentence(message: string) {
  return message ? `${message.charAt(0).toUpperCase()}${message.slice(1)}.` : ''
}

// useNFeImportActions holds the "Importar XML" and "Buscar pela chave"
// buttons of the NF-e page: the fetch dialog and the notifications. The
// requests go through useNFeDocuments, which keeps their markers.
export function useNFeImportActions(options: NFeImportActionsOptions) {
  const $q = useQuasar()
  const { notifyError, notifySuccess, notifyInfo, notifyWarning, notifySyncError } = useNotify()

  async function importXML() {
    try {
      const result = await options.importXML()
      if (result) notifyImportResult(result)
    } catch (error) {
      notifyError('Erro ao importar XML', error)
    }
  }

  // notifyImportResult counts the files and, when some were rejected, lists
  // every file with its reason.
  function notifyImportResult(result: ImportXMLResult) {
    const notify = result.Rejected > 0 ? notifyWarning : notifySuccess
    notify(`Importação: ${importSummary(result)}.`)
    if (result.Rejected > 0) {
      $q.dialog({ component: ImportResultDialog, componentProps: { result, source: 'nfe' } })
    }
  }

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
        notifyInfo(sentence(result.Message) || 'A SEFAZ entregou só o resumo da NF-e.')
        return
      default:
        notifyWarning(sentence(result.Message) || 'A SEFAZ não encontrou a NF-e.')
    }
  }

  return { importXML, openFetchDialog }
}
