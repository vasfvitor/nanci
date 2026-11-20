import { computed, type Ref } from 'vue'
import { useQuasar } from 'quasar'
import ImportResultDialog from '@/components/ImportResultDialog.vue'
import { useNotify } from '@/composables/useNotify'
import type { ImportXMLResult } from '@/types/desktop'
import { importSummary } from '@/utils/importDisplay'

export type ImportActionsOptions = {
  cnpj: Readonly<Ref<string>>
  exporting: Readonly<Ref<boolean>>
  isSyncing: Readonly<Ref<boolean>>
  isResetting: Readonly<Ref<boolean>>
  // importXML returns null when it did not run.
  importXML: () => Promise<ImportXMLResult | null>
}

// useImportActions holds the "Importar XML" button of the NF-e and CT-e
// pages: when it is disabled and the notifications. The request goes through
// the source's documents composable, which keeps its marker.
export function useImportActions(source: 'nfe' | 'cte', options: ImportActionsOptions) {
  const $q = useQuasar()
  const { notifyError, notifySuccess, notifyWarning } = useNotify()

  // The import holds the source's sync marker, so it waits for a pull, a
  // reset or an export to finish.
  const importDisabled = computed(
    () =>
      options.exporting.value ||
      options.isSyncing.value ||
      options.isResetting.value ||
      !options.cnpj.value
  )

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
      $q.dialog({ component: ImportResultDialog, componentProps: { result, source } })
    }
  }

  return { importDisabled, importXML }
}
