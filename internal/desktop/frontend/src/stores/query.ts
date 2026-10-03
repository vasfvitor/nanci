import { ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'

export type QueryType = 'nfse' | 'events'

export const useQueryStore = defineStore('query', () => {
  // The company of a query is the workspace one; the form keeps the chave.
  const form = ref({
    chave: '',
  })
  const result = shallowRef('')
  const type = shallowRef<QueryType>('nfse')
  const loading = shallowRef(false)

  function clearResult() {
    result.value = ''
  }

  return {
    form,
    result,
    type,
    loading,
    clearResult,
  }
})
