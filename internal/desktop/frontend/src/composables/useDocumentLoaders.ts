// LoaderStore is the part of a page store the loaders read and fill.
export type LoaderStore<Input extends { CNPJ: string }, Row> = {
  // filter.CNPJ is the company the page shows now.
  filter: { CNPJ: string }
  // listInput is the list request built from the current filter.
  readonly listInput: Input
  // listError explains why listInput must not be sent, such as a malformed
  // field the page already flags, or is ''.
  readonly listError?: string
  loading: boolean
  setRows: (rows: Row[]) => void
}

// useDocumentLoaders loads a document list into a page store and, for the
// sources with a sync status, the status through fetchStatus into
// store.status. A result that arrives after the user picked another company
// is dropped.
export function useDocumentLoaders<Input extends { CNPJ: string }, Row, Status = never>(
  store: LoaderStore<Input, Row> & { status?: Status | null },
  list: (input: Input) => Promise<Row[]>,
  fetchStatus?: (cnpj: string) => Promise<Status>
) {
  const isSelected = (cnpj: string) => store.filter.CNPJ === cnpj

  async function search(): Promise<Row[]> {
    const input = store.listInput
    if (!input.CNPJ || store.listError) return []
    store.loading = true
    try {
      const result = await list(input)
      if (isSelected(input.CNPJ)) store.setRows(result)
      return result
    } finally {
      store.loading = false
    }
  }

  async function loadStatus(cnpj: string = store.filter.CNPJ): Promise<Status | null> {
    if (!fetchStatus) return null
    if (!cnpj) {
      store.status = null
      return null
    }
    const result = await fetchStatus(cnpj)
    if (isSelected(cnpj)) store.status = result
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
