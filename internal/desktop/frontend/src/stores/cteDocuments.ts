import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { documentListState } from '@/stores/documentListState'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CTeRow, CTeStatusResult, ListCTeInput } from '@/types/desktop'
import { latestOnly } from '@/utils/latestOnly'

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
  const workspace = useWorkspaceStore()

  // filter holds the CT-e filters; the company and the competência come from
  // the workspace.
  const filter = ref<Omit<ListCTeInput, 'CNPJ' | 'Competence'>>({
    Situacao: '',
    Role: '',
    Modelo: '',
    EmitenteCNPJ: '',
    TomadorCNPJ: '',
    NFeChave: '',
    OnlyUnread: false,
  })

  // listInput is the only place the ListCTe request is built from the
  // workspace and the filter. Clearable inputs set null, so every field is
  // normalized here.
  const listInput = computed<ListCTeInput>(() => ({
    CNPJ: workspace.cnpj,
    Competence: workspace.competence,
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

  const status = shallowRef<CTeStatusResult | null>(null)
  // statusGate lets only the latest status load fill status.
  const statusGate = latestOnly()

  return {
    filter,
    listInput,
    listError,
    ...documentListState<CTeRow>(),
    status,
    statusGate,
  }
})
