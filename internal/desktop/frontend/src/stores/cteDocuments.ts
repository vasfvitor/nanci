import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { CTeRow, CTeStatusResult, ListCTeInput } from '@/types/desktop'

// compactCode keeps the letters and digits of a typed CNPJ or access key,
// uppercased, so a pasted "12.345.678/0001-00" or a key in groups of 4 match
// as the backend stores them. CNPJs, and so access keys, may carry letters
// (IN RFB 2.229/2024). Clearable inputs set null, which reads as ''.
function compactCode(value: string | null) {
  return (value ?? '').replace(/[^0-9A-Za-z]/g, '').toUpperCase()
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
  }))

  const rows = ref<CTeRow[]>([])
  const loading = shallowRef(false)
  const exporting = shallowRef(false)
  // incremental makes the ZIP export skip the CT-e already exported.
  const incremental = shallowRef(false)
  const status = shallowRef<CTeStatusResult | null>(null)
  // resettingCNPJ is the company whose CT-e reset is in flight, or ''.
  const resettingCNPJ = shallowRef('')

  return {
    filter,
    listInput,
    rows,
    loading,
    exporting,
    incremental,
    status,
    resettingCNPJ,
  }
})
