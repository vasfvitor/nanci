import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { DocumentRow, ListDocumentsInput } from '@/types/desktop'
import { pruneSelection } from '@/utils/selection'

// The store holds NFS-e page state that outlives the page.
export const useDocumentsStore = defineStore('documents', () => {
  const filter = ref<ListDocumentsInput>({
    CNPJ: '',
    Competence: '',
    Direction: '',
    OnlyUnread: false,
  })

  // listInput is the only place the ListDocuments request is built from the
  // filter. Clearable inputs set null, so every field is normalized here.
  const listInput = computed<ListDocumentsInput>(() => ({
    CNPJ: filter.value.CNPJ || '',
    Competence: filter.value.Competence || '',
    Direction: filter.value.Direction || '',
    OnlyUnread: filter.value.OnlyUnread || false,
  }))

  const documents = ref<DocumentRow[]>([])
  const selected = ref<DocumentRow[]>([])
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

  return {
    filter,
    listInput,
    documents,
    selected,
    filterText,
    loading,
    exporting,
    markingViewed,
    resettingCNPJ,
    setRows,
  }
})
