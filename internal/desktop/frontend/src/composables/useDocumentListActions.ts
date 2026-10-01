import type { Ref } from 'vue'
import { useQuasar } from 'quasar'
import ExportDialog, { type ExportChoice } from '@/components/ExportDialog.vue'
import { useNotify } from '@/composables/useNotify'
import { agree, DOCUMENT_SOURCES, type DocumentSource } from '@/utils/documentSources'

export type DocumentListActionsOptions = {
  source: DocumentSource
  selected: Ref<unknown[]>
  // scopeRows are the documents the actions act on: the selection, or else
  // every row the grid shows.
  scopeRows: Ref<{ ChaveAcesso: string }[]>
  unviewedChaves: Ref<string[]>
  // markViewed returns how many documents were new, or null when nothing
  // was sent.
  markViewed: (chavesAcesso: string[]) => Promise<number | null>
  exportFormats: { label: string; value: string }[]
  showIncludeResumos?: boolean
  // exportList runs the export the user chose in the dialog and reports it.
  exportList: (chavesAcesso: string[], choice: ExportChoice) => Promise<void>
}

// markViewedQuestion asks to mark the count new documents of the list:
// "Marcar como vistas as 3 NF-e novas da lista?".
function markViewedQuestion(source: DocumentSource, count: number) {
  const { noun } = DOCUMENT_SOURCES[source]
  const article = agree(source, '', count)
  const amount = count === 1 ? '' : `${count} `
  return `Marcar como ${agree(source, 'vist', count)} ${article} ${amount}${noun} ${agree(source, 'nov', count)} da lista?`
}

// useDocumentListActions holds the "Marcar vistos" and "Exportar" buttons of
// a document page: the confirmation, the export dialog and the
// notifications. The requests go through the page's composable.
export function useDocumentListActions(options: DocumentListActionsOptions) {
  const $q = useQuasar()
  const { notifyError, notifySuccess } = useNotify()
  const { noun } = DOCUMENT_SOURCES[options.source]

  // confirmMarkViewed marks the new documents of the selection right away;
  // for the whole list it asks first.
  function confirmMarkViewed() {
    const chaves = options.unviewedChaves.value
    if (chaves.length === 0) return
    if (options.selected.value.length > 0) {
      void markViewed(chaves)
      return
    }
    $q.dialog({
      title: 'Marcar vistos',
      message: markViewedQuestion(options.source, chaves.length),
      cancel: true,
      persistent: true,
      ok: { label: 'Marcar vistos', color: 'primary' },
    }).onOk(() => {
      void markViewed(chaves)
    })
  }

  async function markViewed(chaves: string[]) {
    let count: number | null
    try {
      count = await options.markViewed(chaves)
    } catch (error) {
      notifyError(`Erro ao marcar ${noun} como ${agree(options.source, 'vist', 2)}`, error)
      return
    }
    if (count === null) return
    notifySuccess(count === 1 ? '1 documento marcado como visto.' : `${count} documentos marcados como vistos.`)
  }

  // openExportDialog exports the scope rows in the format the user picks.
  function openExportDialog() {
    const chaves = options.scopeRows.value.map((row) => row.ChaveAcesso)
    if (chaves.length === 0) return
    $q.dialog({
      component: ExportDialog,
      componentProps: {
        source: options.source,
        count: chaves.length,
        scope: options.selected.value.length > 0 ? 'selected' : 'listed',
        formats: options.exportFormats,
        showIncludeResumos: options.showIncludeResumos ?? false,
      },
    }).onOk((choice: ExportChoice) => {
      void options.exportList(chaves, choice)
    })
  }

  return { confirmMarkViewed, openExportDialog }
}
