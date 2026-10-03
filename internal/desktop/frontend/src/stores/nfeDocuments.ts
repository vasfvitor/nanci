import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { ListNFeInput, NFePendingRow, NFeRow, NFeStatusResult } from '@/types/desktop'
import { useWorkspaceStore, type WorkspaceKey } from '@/stores/workspace'
import { pruneSelection } from '@/utils/selection'

export type NFeTab = 'notas' | 'pendencias'

// The store holds NF-e page state that outlives the page. Composables write
// plain state through storeToRefs; only setRows and patchRow carry logic.
export const useNFeDocumentsStore = defineStore('nfeDocuments', () => {
  const workspace = useWorkspaceStore()

  // filter holds the NF-e filters; the company and the competência come from
  // the workspace.
  const filter = ref<Omit<ListNFeInput, 'CNPJ' | 'Competence'>>({
    Situacao: '',
    Completeness: '',
    Manifestacao: '',
    Role: '',
    EmitenteCNPJ: '',
    OnlyUnread: false,
  })

  // listInput is the only place the ListNFe request is built from the
  // workspace and the filter. Clearable inputs set null, so every field is
  // normalized here.
  const listInput = computed<ListNFeInput>(() => ({
    CNPJ: workspace.cnpj,
    Competence: workspace.competence || '',
    Situacao: filter.value.Situacao || '',
    Completeness: filter.value.Completeness || '',
    Manifestacao: filter.value.Manifestacao || '',
    Role: filter.value.Role || '',
    EmitenteCNPJ: filter.value.EmitenteCNPJ || '',
    OnlyUnread: Boolean(filter.value.OnlyUnread),
  }))

  const rows = ref<NFeRow[]>([])
  const selected = ref<NFeRow[]>([])
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
  const status = shallowRef<NFeStatusResult | null>(null)
  // statusSeq numbers the status loads; only the latest one fills status.
  const statusSeq = shallowRef(0)
  const activeTab = shallowRef<NFeTab>('notas')
  const pending = ref<NFePendingRow[]>([])
  const pendingLoading = shallowRef(false)
  // pendingSeq numbers the pendências loads; only the latest one fills
  // pending.
  const pendingSeq = shallowRef(0)
  // planningCiencia is true while the backend plans a ciência.
  const planningCiencia = shallowRef(false)
  // cienciaInFlight holds the chaves of the ciência being sent, or null.
  const cienciaInFlight = shallowRef<string[] | null>(null)
  // manifestacaoInFlight holds the chaves whose conclusive manifestação is
  // being sent.
  const manifestacaoInFlight = ref(new Set<string>())
  // resettingCNPJ is the company whose NF-e reset is in flight, or ''.
  const resettingCNPJ = shallowRef('')

  // setRows replaces the result set and keeps only the selected notes that are
  // still present, swapped for their fresh rows so eligibility is current.
  function setRows(next: NFeRow[]) {
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

  // patchRow swaps the note with chave for its fresh row, or drops it when
  // fresh is null because the note no longer matches the filters. The
  // selection follows the same way.
  function patchRow(chave: string, fresh: NFeRow | null) {
    const patch = (list: NFeRow[]) =>
      list.flatMap((row) => {
        if (row.ChaveAcesso !== chave) return [row]
        return fresh ? [fresh] : []
      })
    rows.value = patch(rows.value)
    selected.value = patch(selected.value)
  }

  // busyChaves are the notes with an event being sent.
  const busyChaves = computed(
    () => new Set([...(cienciaInFlight.value ?? []), ...manifestacaoInFlight.value])
  )

  function isChaveBusy(chave: string) {
    return busyChaves.value.has(chave)
  }

  return {
    filter,
    listInput,
    rows,
    selected,
    rowsFor,
    searchSeq,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    statusSeq,
    activeTab,
    pending,
    pendingLoading,
    pendingSeq,
    planningCiencia,
    cienciaInFlight,
    manifestacaoInFlight,
    resettingCNPJ,
    setRows,
    clearRows,
    patchRow,
    isChaveBusy,
  }
})
