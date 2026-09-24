import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { ListNFeInput, NFePendingRow, NFeRow, NFeStatusResult } from '@/types/desktop'

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
  }))

  const rows = ref<NFeRow[]>([])
  const selected = ref<NFeRow[]>([])
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
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
    if (selected.value.length === 0) return

    const byChave = new Map(next.map((row) => [row.ChaveAcesso, row]))
    selected.value = selected.value.flatMap((row) => {
      const fresh = byChave.get(row.ChaveAcesso)
      return fresh ? [fresh] : []
    })
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
    loading,
    exporting,
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
