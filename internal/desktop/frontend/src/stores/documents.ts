import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { documentListState } from '@/stores/documentListState'
import { useWorkspaceStore } from '@/stores/workspace'
import type { DocumentRow, ListDocumentsInput } from '@/types/desktop'

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
    Competence: workspace.competence,
    Direction: filter.value.Direction || '',
    OnlyUnread: filter.value.OnlyUnread || false,
  }))

  return {
    filter,
    listInput,
    ...documentListState<DocumentRow>(),
  }
})
