import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { ListNFeInput, NFePendingRow, NFeRow, NFeStatusResult } from '@/types/desktop'
import { pruneSelection } from '@/utils/selection'

export type NFeTab = 'notas' | 'pendencias'

// The store holds NF-e page state that outlives the page. Composables write
// plain state through storeToRefs; only setRows and patchRow carry logic.
export const useNFeDocumentsStore = defineStore('nfeDocuments', () => {
  const filter = ref<ListNFeInput>({
    CNPJ: '',
    Competence: '',
    Situacao: '',
    Completeness: '',
    Manifestacao: '',
    Role: '',
    EmitenteCNPJ: '',
    OnlyUnread: false,
  })

  // listInput is the only place the ListNFe request is built from the filter.
  // Clearable inputs set null, so every field is normalized here.
  const listInput = computed<ListNFeInput>(() => ({
    CNPJ: filter.value.CNPJ || '',
    Competence: filter.value.Competence || '',
    Situacao: filter.value.Situacao || '',
    Completeness: filter.value.Completeness || '',
    Manifestacao: filter.value.Manifestacao || '',
    Role: filter.value.Role || '',
    EmitenteCNPJ: filter.value.EmitenteCNPJ || '',
    OnlyUnread: Boolean(filter.value.OnlyUnread),
  }))

  const rows = ref<NFeRow[]>([])
  const selected = ref<NFeRow[]>([])
  // filterText narrows the listed rows on the page, without a new search.
  const filterText = ref('')
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
  // markingViewed is true while a "Marcar vistos" request is in flight.
  const markingViewed = shallowRef(false)
  const status = shallowRef<NFeStatusResult | null>(null)
  const activeTab = shallowRef<NFeTab>('notas')
  const pending = ref<NFePendingRow[]>([])
  const pendingLoading = shallowRef(false)
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

  function isChaveBusy(chave: string) {
    return Boolean(cienciaInFlight.value?.includes(chave)) || manifestacaoInFlight.value.has(chave)
  }

  return {
    filter,
    listInput,
    rows,
    selected,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    activeTab,
    pending,
    pendingLoading,
    planningCiencia,
    cienciaInFlight,
    manifestacaoInFlight,
    resettingCNPJ,
    setRows,
    patchRow,
    isChaveBusy,
  }
})
