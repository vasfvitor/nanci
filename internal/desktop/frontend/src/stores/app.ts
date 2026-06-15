import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  // Common App State
  const activeCompanyId = ref<string | null>(null)
  
  // Cache for the Query Page
  const queryForm = ref({
    cnpj: '',
    chave: ''
  })
  const queryResult = ref('')
  const queryType = ref('nfse')

  return {
    activeCompanyId,
    queryForm,
    queryResult,
    queryType
  }
})
