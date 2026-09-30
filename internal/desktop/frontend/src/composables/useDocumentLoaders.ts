import type { Ref } from 'vue'

export type DocumentLoadersOptions<Input extends { CNPJ: string }, Row, Status> = {
  // selectedCNPJ is the company the page shows now.
  selectedCNPJ: () => string
  // listInput is the list request built from the current filter.
  listInput: () => Input
  // canList rejects a request that must not be sent, such as one with a
  // malformed field the page already flags.
  canList?: (input: Input) => boolean
  list: (input: Input) => Promise<Row[]>
  setRows: (rows: Row[]) => void
  loading: Ref<boolean>
  // status is the company's sync status, for sources that have one.
  status?: {
    state: Ref<Status | null>
    fetch: (cnpj: string) => Promise<Status>
  }
}

// useDocumentLoaders loads a document list, and the sync status when there
// is one, into a page store. A result that arrives after the user picked
// another company is dropped.
export function useDocumentLoaders<Input extends { CNPJ: string }, Row, Status = never>(
  options: DocumentLoadersOptions<Input, Row, Status>
) {
  const isSelected = (cnpj: string) => options.selectedCNPJ() === cnpj

  async function search(): Promise<Row[]> {
    const input = options.listInput()
    if (!input.CNPJ || options.canList?.(input) === false) return []
    options.loading.value = true
    try {
      const result = await options.list(input)
      if (isSelected(input.CNPJ)) options.setRows(result)
      return result
    } finally {
      options.loading.value = false
    }
  }

  async function loadStatus(cnpj: string = options.selectedCNPJ()): Promise<Status | null> {
    const status = options.status
    if (!status) return null
    if (!cnpj) {
      status.state.value = null
      return null
    }
    const result = await status.fetch(cnpj)
    if (isSelected(cnpj)) status.state.value = result
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

  return { isSelected, search, loadStatus, refresh }
}
