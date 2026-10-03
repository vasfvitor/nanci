import { ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { latestOnly } from '@/utils/latestOnly'

export type QueryType = 'nfse' | 'events'

// ChaveOption is a chave suggestion of the query form.
export type ChaveOption = { label: string; value: string; description?: string }

export const useQueryStore = defineStore('query', () => {
  // The company of a query is the workspace one; the form keeps the chave.
  const form = ref({
    chave: '',
  })
  const result = shallowRef('')
  const type = shallowRef<QueryType>('nfse')
  const loading = shallowRef(false)
  // chaveOptions are the chave suggestions listed for optionsCNPJ, or [] with
  // an empty optionsCNPJ while none are listed.
  const chaveOptions = shallowRef<ChaveOption[]>([])
  const optionsCNPJ = shallowRef('')
  // optionsGate lets only the latest suggestion fetch fill chaveOptions.
  const optionsGate = latestOnly()

  function clearResult() {
    result.value = ''
  }

  return {
    form,
    result,
    type,
    loading,
    chaveOptions,
    optionsCNPJ,
    optionsGate,
    clearResult,
  }
})
