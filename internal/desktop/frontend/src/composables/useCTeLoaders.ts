import { storeToRefs } from 'pinia'
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
export function useCTeLoaders() {
  const store = useCTeDocumentsStore()
  const { loading, status } = storeToRefs(store)

  const isSelected = (cnpj: string) => store.filter.CNPJ === cnpj

  // search lists the CT-e of the filter. An NF-e key that is not 44
  // characters is never sent; the page shows the error on the field.
  async function search() {
    const input = store.listInput
    if (!input.CNPJ || !isNFeChaveFilter(input.NFeChave)) return []
    loading.value = true
    try {
      const result = await desktopClient.listCTe(input)
      if (isSelected(input.CNPJ)) store.rows = result
      return result
    } finally {
      loading.value = false
    }
  }

  async function loadStatus(cnpj: string = store.filter.CNPJ) {
    if (!cnpj) {
      status.value = null
      return null
    }
    const result = await desktopClient.statusCTe(cnpj)
    if (isSelected(cnpj)) status.value = result
    return result
  }

  // refresh reloads the list and status after work that already happened: a
  // sync or a reset. It also runs after a failed pull, because the status
  // then carries the block reason. Its own failures must not hide the result
  // of that work.
  async function refresh(cnpj: string) {
    if (!isSelected(cnpj)) return
    await Promise.allSettled([search(), loadStatus(cnpj)])
  }

  return { search, loadStatus, refresh }
}
