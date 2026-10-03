import type { WorkspaceKey } from '@/stores/workspace'
import { latestOnly, type LatestGate } from '@/utils/latestOnly'

// LoaderStore is the part of a page store the loaders read and fill.
export type LoaderStore<Input extends { CNPJ: string; Competence: string }, Row> = {
  // listInput is the list request built from the workspace and the filter;
  // its CNPJ is the company the page shows now.
  readonly listInput: Input
  // listError explains why listInput must not be sent, such as a malformed
  // field the page already flags, or is ''.
  readonly listError?: string
  loading: boolean
  // rowsFor is the company and competência of the search that filled the
  // rows, or null.
  rowsFor: WorkspaceKey | null
  // searchGate lets only the latest search fill the rows.
  readonly searchGate: LatestGate
  setRows: (rows: Row[]) => void
}

// StatusStore is the part of a page store that holds a sync status. A page
// without one leaves both fields out.
export type StatusStore<Status> = {
  status: Status | null
  // statusGate lets only the latest status load fill status.
  readonly statusGate: LatestGate
}

// useDocumentLoaders loads the document list of a page into its store and,
// when the source has a sync status (fetchStatus), the status into
// store.status. A result that arrives after the user picked another company
// is dropped, and so is a list or a status from a load that a newer one
// replaced.
export function useDocumentLoaders<
  Input extends { CNPJ: string; Competence: string },
  Row,
  Status = never,
>(
  store: LoaderStore<Input, Row> & Partial<StatusStore<Status>>,
  list: (input: Input) => Promise<Row[]>,
  fetchStatus?: (cnpj: string) => Promise<Status>
) {
  const isSelected = (cnpj: string) => store.listInput.CNPJ === cnpj
  // A store without a status has no statusGate; loadStatus then returns
  // before using this one.
  const statusGate = store.statusGate ?? latestOnly()

  async function search(): Promise<Row[]> {
    const input = store.listInput
    if (!input.CNPJ || store.listError) return []
    // Quick changes, such as stepping through competências, start searches
    // that may end out of order; the latest one owns the rows and loading.
    const isLatest = store.searchGate.begin()
    store.loading = true
    try {
      const result = await list(input)
      if (isLatest() && isSelected(input.CNPJ)) {
        store.setRows(result)
        store.rowsFor = { cnpj: input.CNPJ, competence: input.Competence }
      }
      return result
    } finally {
      if (isLatest()) store.loading = false
    }
  }

  async function loadStatus(cnpj: string = store.listInput.CNPJ): Promise<Status | null> {
    if (!fetchStatus) return null
    // A sync, a reset and a page visit may load the status of the same
    // company at once; the latest load owns it.
    const isLatest = statusGate.begin()
    if (!cnpj) {
      store.status = null
      return null
    }
    const result = await fetchStatus(cnpj)
    if (isLatest() && isSelected(cnpj)) store.status = result
    return result
  }

  // refresh reloads the list and the status after a sync or a reset, failed
  // pulls included, since the status then carries the block reason. Its own
  // failures are not reported, so they never hide the result of the sync.
  async function refresh(cnpj: string) {
    if (!isSelected(cnpj)) return
    await Promise.allSettled([search(), loadStatus(cnpj)])
  }

  return { isSelected, search, loadStatus, refresh }
}
