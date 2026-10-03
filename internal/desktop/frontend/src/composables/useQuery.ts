import { shallowRef, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { type ChaveOption, useQueryStore } from '@/stores/query'
import { useWorkspaceStore } from '@/stores/workspace'
import type { DocumentRow } from '@/types/desktop'
import { formatCurrencyCents } from '@/utils/formatters'

// chaveOption is an NFS-e as a chave suggestion: the 50 digits, and the
// last digits, the other party and the value to recognize it by.
function chaveOption(row: DocumentRow): ChaveOption {
  const pureKey = row.ChaveAcesso.replace(/\D/g, '')
  const shortKey = pureKey.length >= 6 ? pureKey.slice(-6) : pureKey
  const name = row.CompanyRole === 'prestada' ? row.TomadorName : row.PrestadorName
  const shortName = name && name.length > 20 ? name.slice(0, 20) + '...' : name
  return {
    label: pureKey,
    value: pureKey,
    description: `...${shortKey} | ${shortName || 'Desconhecido'} | ${formatCurrencyCents(row.ServiceValue)}`,
  }
}

export function useQuery() {
  const queryStore = useQueryStore()
  const { form, result, type, loading, chaveOptions, optionsCNPJ } = storeToRefs(queryStore)
  // The query authenticates as the workspace company.
  const workspace = useWorkspaceStore()
  const { cnpj } = storeToRefs(workspace)

  // documentOptions are the chave suggestions narrowed by filterDocuments.
  const documentOptions = shallowRef<ChaveOption[]>(chaveOptions.value)

  // The chave suggestions are the NFS-e of the workspace company. They stay
  // in the query store for the company they were listed for, so opening the
  // page again lists them only after a company change. A fetch that ends
  // after the company changed again is dropped.
  watch(
    cnpj,
    async (newCnpj) => {
      const isLatest = queryStore.optionsGate.begin()
      if (newCnpj === optionsCNPJ.value) {
        documentOptions.value = chaveOptions.value
        return
      }
      chaveOptions.value = []
      optionsCNPJ.value = ''
      documentOptions.value = []
      if (!newCnpj) return
      try {
        const docs = await desktopClient.listDocuments({
          CNPJ: newCnpj,
          Competence: '',
          Direction: '',
          OnlyUnread: false,
        })
        if (!isLatest()) return
        chaveOptions.value = docs.map(chaveOption)
        optionsCNPJ.value = newCnpj
        documentOptions.value = chaveOptions.value
      } catch {
        // The suggestions are optional; the chave can still be typed.
      }
    },
    { immediate: true }
  )

  function filterDocuments(val: string) {
    if (val === '') {
      documentOptions.value = chaveOptions.value
      return
    }
    const needle = val.toLowerCase()
    documentOptions.value = chaveOptions.value.filter(
      (v) => v.value.toLowerCase().includes(needle) || (v.description && v.description.toLowerCase().includes(needle))
    )
  }

  async function runQuery() {
    if (loading.value) return result.value
    
    const chaveVal = typeof form.value.chave === 'object' && form.value.chave !== null 
      ? (form.value.chave as { value: string }).value 
      : form.value.chave

    const companyCNPJ = cnpj.value
    if (!companyCNPJ || !/^\d{50}$/.test(chaveVal)) return ''

    loading.value = true
    queryStore.clearResult()
    try {
      const input = {
        CompanyCNPJ: companyCNPJ,
        ChaveAcesso: chaveVal,
      }
      result.value = await desktopClient.queryNFSeEvents(input)
      return result.value
    } finally {
      loading.value = false
    }
  }

  return {
    form,
    result,
    type,
    loading,
    documentOptions,
    filterDocuments,
    runQuery,
  }
}
