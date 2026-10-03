import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { CTeRow, CTeStatusResult, ListCTeInput } from '@/types/desktop'
import type { WorkspaceKey } from '@/stores/workspace'
import { pruneSelection } from '@/utils/selection'

// compactCode keeps the letters and digits of a typed CNPJ or access key,
// uppercased, so a pasted "12.345.678/0001-00" or a key in groups of 4 match
// as the backend stores them. CNPJs, and so access keys, may carry letters
// (IN RFB 2.229/2024). Clearable inputs set null, which reads as ''.
function compactCode(value: string | null) {
  return (value ?? '').replace(/[^0-9A-Za-z]/g, '').toUpperCase()
}

// isNFeChaveFilter accepts an empty NF-e filter or a 44-character key. The
// emitente CNPJ inside the key may carry letters, so they are allowed; the
// backend checks the rest.
function isNFeChaveFilter(chave: string) {
  return chave === '' || /^[0-9A-Z]{44}$/.test(chave)
}

// The store holds CT-e page state that outlives the page.
export const useCTeDocumentsStore = defineStore('cteDocuments', () => {
  const filter = ref<ListCTeInput>({
    CNPJ: '',
    Competence: '',
    Situacao: '',
    Role: '',
    Modelo: '',
    EmitenteCNPJ: '',
    TomadorCNPJ: '',
    NFeChave: '',
    OnlyUnread: false,
  })

  // listInput is the only place the ListCTe request is built from the filter.
  // Clearable inputs set null, so every field is normalized here.
  const listInput = computed<ListCTeInput>(() => ({
    CNPJ: filter.value.CNPJ || '',
    Competence: filter.value.Competence || '',
    Situacao: filter.value.Situacao || '',
    Role: filter.value.Role || '',
    Modelo: filter.value.Modelo || '',
    EmitenteCNPJ: compactCode(filter.value.EmitenteCNPJ),
    TomadorCNPJ: compactCode(filter.value.TomadorCNPJ),
    NFeChave: compactCode(filter.value.NFeChave),
    OnlyUnread: Boolean(filter.value.OnlyUnread),
  }))

  // listError explains why listInput cannot be sent, or is ''. The page
  // shows it on the NF-e key field.
  const listError = computed(() =>
    isNFeChaveFilter(listInput.value.NFeChave) ? '' : 'A chave de NF-e tem 44 caracteres'
  )

  const rows = ref<CTeRow[]>([])
  const selected = ref<CTeRow[]>([])
  // rowsFor is the company and competência of the search that filled the
  // rows, or null when no search did.
  const rowsFor = shallowRef<WorkspaceKey | null>(null)
  // searchSeq numbers the list searches; only the latest one fills the rows.
  const searchSeq = shallowRef(0)
  // filterText narrows the listed rows on the page, without a new search.
  const filterText = ref('')
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
  // markingViewed is true while a "Marcar vistos" request is in flight.
  const markingViewed = shallowRef(false)
  const status = shallowRef<CTeStatusResult | null>(null)
  // resettingCNPJ is the company whose CT-e reset is in flight, or ''.
  const resettingCNPJ = shallowRef('')

  // setRows replaces the result set and keeps only the selected CT-e that are
  // still present, swapped for their fresh rows.
  function setRows(next: CTeRow[]) {
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
    filter,
    listInput,
    listError,
    rows,
    selected,
    rowsFor,
    searchSeq,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    resettingCNPJ,
    setRows,
    clearRows,
  }
})
