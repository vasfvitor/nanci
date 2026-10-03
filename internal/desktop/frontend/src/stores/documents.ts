import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { DocumentRow, ListDocumentsInput } from '@/types/desktop'
import { useWorkspaceStore, type WorkspaceKey } from '@/stores/workspace'
import { pruneSelection } from '@/utils/selection'

// The store holds NFS-e page state that outlives the page.
export const useDocumentsStore = defineStore('documents', () => {
  const workspace = useWorkspaceStore()

  // filter holds the NFS-e filters; the company and the competência come
  // from the workspace.
  const filter = ref<Omit<ListDocumentsInput, 'CNPJ' | 'Competence'>>({
    Direction: '',
    OnlyUnread: false,
  })

  // listInput is the only place the ListDocuments request is built from the
  // workspace and the filter. Clearable inputs set null, so every field is
  // normalized here.
  const listInput = computed<ListDocumentsInput>(() => ({
    CNPJ: workspace.cnpj,
    Competence: workspace.competence || '',
    Direction: filter.value.Direction || '',
    OnlyUnread: filter.value.OnlyUnread || false,
  }))

  const documents = ref<DocumentRow[]>([])
  const selected = ref<DocumentRow[]>([])
  // rowsFor is the company and competência of the search that filled the
  // rows, or null when no search did.
  const rowsFor = shallowRef<WorkspaceKey | null>(null)
  // searchSeq numbers the list searches; only the latest one fills the rows.
  const searchSeq = shallowRef(0)
  // filterText narrows the listed rows on the page, without a new search.
  const filterText = ref('')
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
  // markingViewed is true while a "Marcar vistos" request is in flight.
  const markingViewed = shallowRef(false)
  // resettingCNPJ is the company whose NFS-e sync reset is in flight, or ''.
  const resettingCNPJ = shallowRef('')

  // setRows replaces the result set and keeps only the selected documents
  // that are still present, swapped for their fresh rows.
  function setRows(next: DocumentRow[]) {
    documents.value = next
    selected.value = pruneSelection(next, selected.value)
  }

  // clearRows empties the result set and the selection, for a page that
  // must not show rows of another company or competência.
  function clearRows() {
    documents.value = []
    selected.value = []
    rowsFor.value = null
  }

  return {
    filter,
    listInput,
    documents,
    selected,
    rowsFor,
    searchSeq,
    filterText,
    loading,
    exporting,
    markingViewed,
    resettingCNPJ,
    setRows,
    clearRows,
  }
})
