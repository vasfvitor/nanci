import type { Ref } from 'vue'
import { useQuasar } from 'quasar'
import ExportDialog, { type ExportChoice, type ExportNoun } from '@/components/ExportDialog.vue'
import type { MarkViewedResult } from '@/composables/useMarkViewed'
import { useNotify } from '@/composables/useNotify'

export type DocumentListActionsOptions = {
  noun: ExportNoun
  selected: Ref<unknown[]>
  // scopeRows are the documents the actions act on: the selection, or else
  // every row the grid shows.
  scopeRows: Ref<{ ChaveAcesso: string }[]>
  unviewedChaves: Ref<string[]>
  markViewed: (chavesAcesso: string[]) => Promise<MarkViewedResult | null>
  exportFormats: { label: string; value: string }[]
  showIncludeResumos?: boolean
  // exportList runs the export the user chose in the dialog and reports it.
  exportList: (chavesAcesso: string[], choice: ExportChoice) => Promise<void>
}

// The NFS-e and the NF-e are feminine, the CT-e (o conhecimento) masculine.
const markViewedText: Record<ExportNoun, { confirm: (count: number) => string; error: string }> = {
  'NFS-e': {
    confirm: (count) =>
      count === 1
        ? 'Marcar como vista a NFS-e nova da lista?'
        : `Marcar como vistas as ${count} NFS-e novas da lista?`,
    error: 'Erro ao marcar NFS-e como vistas',
  },
  'NF-e': {
    confirm: (count) =>
      count === 1
        ? 'Marcar como vista a NF-e nova da lista?'
        : `Marcar como vistas as ${count} NF-e novas da lista?`,
    error: 'Erro ao marcar NF-e como vistas',
  },
  'CT-e': {
    confirm: (count) =>
      count === 1
        ? 'Marcar como visto o CT-e novo da lista?'
        : `Marcar como vistos os ${count} CT-e novos da lista?`,
    error: 'Erro ao marcar CT-e como vistos',
  },
}

// useDocumentListActions holds the "Marcar vistos" and "Exportar" buttons of
// a document page: the confirmation, the export dialog and the
// notifications. The requests go through the page's composable.
export function useDocumentListActions(options: DocumentListActionsOptions) {
  const $q = useQuasar()
  const { notifyError, notifySuccess } = useNotify()
  const text = markViewedText[options.noun]

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
      message: text.confirm(chaves.length),
      cancel: true,
      persistent: true,
      ok: { label: 'Marcar vistos', color: 'primary' },
    }).onOk(() => {
      void markViewed(chaves)
    })
  }

  // markViewed reports the marking first; a list that fails to reload
  // afterwards is reported apart, since the marking went through.
  async function markViewed(chaves: string[]) {
    let result: MarkViewedResult | null
    try {
      result = await options.markViewed(chaves)
    } catch (error) {
      notifyError(text.error, error)
      return
    }
    if (!result) return
    notifySuccess(
      result.count === 1
        ? '1 documento marcado como visto.'
        : `${result.count} documentos marcados como vistos.`
    )
    if (result.reloadError !== null) {
      notifyError('Erro ao recarregar a lista', result.reloadError)
    }
  }

  // openExportDialog exports the scope rows in the format the user picks.
  function openExportDialog() {
    const chaves = options.scopeRows.value.map((row) => row.ChaveAcesso)
    if (chaves.length === 0) return
    $q.dialog({
      component: ExportDialog,
      componentProps: {
        noun: options.noun,
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
