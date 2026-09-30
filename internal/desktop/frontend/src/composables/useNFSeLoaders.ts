import { storeToRefs } from 'pinia'
import { useDocumentLoaders } from '@/composables/useDocumentLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useDocumentsStore } from '@/stores/documents'

// useNFSeLoaders loads the NFS-e list into the documents store. A result
// that arrives after the user picked another company is dropped. The NFS-e
// has no status call: its sync state comes with the company list.
export function useNFSeLoaders() {
  const store = useDocumentsStore()
  const { loading } = storeToRefs(store)

  const { isSelected, search } = useDocumentLoaders({
    selectedCNPJ: () => store.filter.CNPJ,
    listInput: () => store.listInput,
    list: (input) => desktopClient.listDocuments(input),
    setRows: (rows) => store.setRows(rows),
    loading,
  })

  return { isSelected, search }
}
