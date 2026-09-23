import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type {
  ListNFeInput,
  NFeConclusiveTipo,
  NFeEventBatchResult,
  NFePendingRow,
  NFeRow,
  NFeStatusResult,
} from '@/types/desktop'

export type NFeTab = 'notas' | 'pendencias'

export type CienciaInFlight = {
  cnpj: string
  chaves: string[]
}

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
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
  const status = shallowRef<NFeStatusResult | null>(null)
  const activeTab = shallowRef<NFeTab>('notas')
  const pending = ref<NFePendingRow[]>([])
  const pendingLoading = shallowRef(false)
  const cienciaInFlight = shallowRef<CienciaInFlight | null>(null)
  const manifestationInFlight = ref<Record<string, NFeConclusiveTipo>>({})
  const lastCienciaResult = shallowRef<NFeEventBatchResult | null>(null)

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

  function setPending(next: NFePendingRow[]) {
    pending.value = next
  }

  function setStatus(next: NFeStatusResult | null) {
    status.value = next
  }

  function setLastCienciaResult(result: NFeEventBatchResult | null) {
    lastCienciaResult.value = result
  }

  function clearSelection() {
    selected.value = []
  }

  function startCiencia(cnpj: string, chaves: string[]) {
    cienciaInFlight.value = { cnpj, chaves: [...chaves] }
  }

  function finishCiencia() {
    cienciaInFlight.value = null
  }

  function startManifestation(chave: string, tipo: NFeConclusiveTipo) {
    manifestationInFlight.value = { ...manifestationInFlight.value, [chave]: tipo }
  }

  function finishManifestation(chave: string) {
    manifestationInFlight.value = Object.fromEntries(
      Object.entries(manifestationInFlight.value).filter(([key]) => key !== chave)
    )
  }

  function isChaveBusy(chave: string) {
    return (
      Boolean(cienciaInFlight.value?.chaves.includes(chave)) ||
      Boolean(manifestationInFlight.value[chave])
    )
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
    cienciaInFlight,
    manifestationInFlight,
    lastCienciaResult,
    setRows,
    setPending,
    setStatus,
    setLastCienciaResult,
    clearSelection,
    startCiencia,
    finishCiencia,
    startManifestation,
    finishManifestation,
    isChaveBusy,
  }
})
