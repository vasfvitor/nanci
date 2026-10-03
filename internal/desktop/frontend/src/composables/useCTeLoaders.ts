import { useDocumentLoaders } from '@/composables/useDocumentLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'

// useCTeLoaders loads the CT-e list and status into the cteDocuments store.
// A result that arrives after the user picked another company is dropped.
// A list input with a listError, such as an NF-e key that is not 44
// characters, is never sent; the page shows the error on the field.
export function useCTeLoaders() {
  const store = useCTeDocumentsStore()

  const { search, loadStatus, refresh } = useDocumentLoaders(
    store,
    (input) => desktopClient.listCTe(input),
    (cnpj) => desktopClient.statusCTe(cnpj)
  )

  return { search, loadStatus, refresh }
}
