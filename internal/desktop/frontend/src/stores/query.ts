import { ref } from 'vue'
import { defineStore } from 'pinia'

export type QueryType = 'nfse' | 'events'

export const useQueryStore = defineStore('query', () => {
  const form = ref({
    cnpj: '',
    chave: '',
  })
  const result = ref('')
  const type = ref<QueryType>('nfse')

  function clearResult() {
    result.value = ''
  }

  return {
    form,
    result,
    type,
    clearResult,
  }
})
