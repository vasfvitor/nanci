import { storeToRefs } from 'pinia'
import { useDocumentLoaders } from '@/composables/useDocumentLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'

// isNFeChaveFilter accepts an empty NF-e filter or a 44-character key. The
// emitente CNPJ inside the key may carry letters, so they are allowed; the
// backend checks the rest.
export function isNFeChaveFilter(chave: string) {
  return chave === '' || /^[0-9A-Z]{44}$/.test(chave)
}

// useCTeLoaders loads the CT-e list and status into the cteDocuments store.
// A result that arrives after the user picked another company is dropped.
// An NF-e key that is not 44 characters is never sent; the page shows the
// error on the field.
export function useCTeLoaders() {
  const store = useCTeDocumentsStore()
  const { loading, status } = storeToRefs(store)

  const { search, loadStatus, refresh } = useDocumentLoaders({
    selectedCNPJ: () => store.filter.CNPJ,
    listInput: () => store.listInput,
    canList: (input) => isNFeChaveFilter(input.NFeChave),
    list: (input) => desktopClient.listCTe(input),
    setRows: (rows) => store.setRows(rows),
    loading,
    status: { state: status, fetch: (cnpj) => desktopClient.statusCTe(cnpj) },
  })

  return { search, loadStatus, refresh }
}
