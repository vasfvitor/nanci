import { ref, shallowRef, type Ref } from 'vue'
import type { WorkspaceKey } from '@/stores/workspace'
import { latestOnly } from '@/utils/latestOnly'
import { pruneSelection } from '@/utils/selection'

// documentListState is the list state the NFS-e, NF-e and CT-e stores share:
// the listed rows, the selection, the text filter and the busy flags of the
// list actions. Each setup store calls it once and returns its fields.
export function documentListState<Row extends { ChaveAcesso: string }>() {
  const rows = ref([]) as Ref<Row[]>
  const selected = ref([]) as Ref<Row[]>
  // rowsFor is the company and competência of the search that filled the
  // rows, or null when no search did.
  const rowsFor = shallowRef<WorkspaceKey | null>(null)
  // searchGate lets only the latest list search fill the rows; it lives in
  // the store so a remounted page still drops the reply of an earlier search.
  const searchGate = latestOnly()
  // filterText narrows the listed rows on the page, without a new search.
  const filterText = ref('')
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
  // markingViewed is true while a "Marcar vistos" request is in flight.
  const markingViewed = shallowRef(false)
  // resettingCNPJ is the company whose sync reset is in flight, or ''.
  const resettingCNPJ = shallowRef('')

  // setRows replaces the result set and keeps only the selected rows that
  // are still present, swapped for their fresh rows.
  function setRows(next: Row[]) {
    rows.value = next
    selected.value = pruneSelection(next, selected.value)
  }

  // clearRows empties the result set and the selection, for a page that
  // must not show rows of another company or competência.
  function clearRows() {
    rows.value = []
    selected.value = []
    rowsFor.value = null
  }

  return {
    rows,
    selected,
    rowsFor,
    searchGate,
    filterText,
    loading,
    exporting,
    markingViewed,
    resettingCNPJ,
    setRows,
    clearRows,
  }
}
