import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { documentListState } from '@/stores/documentListState'
import { useWorkspaceStore } from '@/stores/workspace'
import type { ListNFeInput, NFePendingRow, NFeRow, NFeStatusResult } from '@/types/desktop'
import { latestOnly } from '@/utils/latestOnly'

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
    Competence: workspace.competence,
    Situacao: filter.value.Situacao || '',
    Completeness: filter.value.Completeness || '',
    Manifestacao: filter.value.Manifestacao || '',
    Role: filter.value.Role || '',
    EmitenteCNPJ: filter.value.EmitenteCNPJ || '',
    OnlyUnread: Boolean(filter.value.OnlyUnread),
  }))

  const list = documentListState<NFeRow>()
  const { rows, selected } = list
  const status = shallowRef<NFeStatusResult | null>(null)
  // statusGate lets only the latest status load fill status.
  const statusGate = latestOnly()
  const activeTab = shallowRef<NFeTab>('notas')
  const pending = ref<NFePendingRow[]>([])
  const pendingLoading = shallowRef(false)
  // pendingGate lets only the latest pendências load fill pending.
  const pendingGate = latestOnly()
  // planningCiencia is true while the backend plans a ciência.
  const planningCiencia = shallowRef(false)
  // cienciaInFlight holds the chaves of the ciência being sent, or null.
  const cienciaInFlight = shallowRef<string[] | null>(null)
  // manifestacaoInFlight holds the chaves whose conclusive manifestação is
  // being sent.
  const manifestacaoInFlight = ref(new Set<string>())

  // patchRow swaps the note with chave for its fresh row, or drops it when
  // fresh is null because the note no longer matches the filters. The
  // selection follows the same way.
  function patchRow(chave: string, fresh: NFeRow | null) {
    const patch = (items: NFeRow[]) =>
      items.flatMap((row) => {
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
    ...list,
    status,
    statusGate,
    activeTab,
    pending,
    pendingLoading,
    pendingGate,
    planningCiencia,
    cienciaInFlight,
    manifestacaoInFlight,
    patchRow,
    isChaveBusy,
  }
})
