import { ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { DocumentRow, ListDocumentsInput } from '@/types/desktop'

export const useDocumentsStore = defineStore('documents', () => {
  const filter = ref<ListDocumentsInput>({
    CNPJ: '',
    Competence: '',
    Direction: '',
    OnlyUnread: false,
  })
  const documents = ref<DocumentRow[]>([])
  const loading = shallowRef(false)
  const exporting = shallowRef(false)

  function setDocuments(rows: DocumentRow[]) {
    documents.value = rows
  }

  function resetDocuments() {
    documents.value = []
  }

  return {
    filter,
    documents,
    loading,
    exporting,
    setDocuments,
    resetDocuments,
  }
})
