import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { DocumentRow, ListDocumentsInput } from '@/types/desktop'

export const useDocumentsStore = defineStore('documents', () => {
  const filter = ref<ListDocumentsInput>({
    CNPJ: '',
    Competence: '',
    Direction: '',
  })
  const documents = ref<DocumentRow[]>([])

  function setDocuments(rows: DocumentRow[]) {
    documents.value = rows
  }

  function resetDocuments() {
    documents.value = []
  }

  return {
    filter,
    documents,
    setDocuments,
    resetDocuments,
  }
})
