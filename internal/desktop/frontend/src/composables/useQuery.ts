import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useQueryStore } from '@/stores/query'

export function useQuery() {
  const queryStore = useQueryStore()
  const { form, result, type, loading } = storeToRefs(queryStore)
  const allOptions = ref<{ label: string; value: string }[]>([])
  const companyOptions = ref<{ label: string; value: string }[]>([])

  const allDocumentOptions = ref<{ label: string; value: string; description?: string }[]>([])
  const documentOptions = ref<{ label: string; value: string; description?: string }[]>([])

  let latestDocumentRequest = 0

  watch(
    () => form.value.cnpj,
    async (newCnpj) => {
      const requestID = ++latestDocumentRequest

      if (!newCnpj) {
        allDocumentOptions.value = []
        documentOptions.value = []
        return
      }
      try {
        const docs = await desktopClient.listDocuments({ CNPJ: newCnpj, Competence: '', Direction: '', OnlyUnread: false })
        if (requestID !== latestDocumentRequest) return

        allDocumentOptions.value = docs.map((d) => {
          const pureKey = d.ChaveAcesso.replace(/\D/g, '')
          const shortKey = pureKey.length >= 6 ? pureKey.slice(-6) : pureKey
          const valStr = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(d.ServiceValue / 100)
          const name = d.CompanyRole === 'prestada' ? d.TomadorName : d.PrestadorName
          const shortName = name && name.length > 20 ? name.slice(0, 20) + '...' : name
          return {
            label: pureKey,
            value: pureKey,
            description: `...${shortKey} | ${shortName || 'Desconhecido'} | ${valStr}`,
          }
        })
        documentOptions.value = allDocumentOptions.value
      } catch {
        // Silently fail autocomplete fetch
      }
    },
    { immediate: true }
  )

  async function loadCompanies() {
    const companies = await desktopClient.listCompanies()
    const options = companies.map((company) => ({
      label: `${company.Name} (${company.CNPJ})`,
      value: company.CNPJ,
    }))
    allOptions.value = options
    companyOptions.value = options
  }

  function filterCompanies(value: string) {
    const needle = value.toLowerCase()
    companyOptions.value = allOptions.value.filter(
      (option) =>
        option.label.toLowerCase().includes(needle) || option.value.includes(needle)
    )
  }

  function filterDocuments(val: string) {
    if (val === '') {
      documentOptions.value = allDocumentOptions.value
      return
    }
    const needle = val.toLowerCase()
    documentOptions.value = allDocumentOptions.value.filter(
      (v) => v.value.toLowerCase().includes(needle) || (v.description && v.description.toLowerCase().includes(needle))
    )
  }

  async function runQuery() {
    if (loading.value) return result.value
    
    const chaveVal = typeof form.value.chave === 'object' && form.value.chave !== null 
      ? (form.value.chave as { value: string }).value 
      : form.value.chave

    if (!form.value.cnpj || !/^\d{50}$/.test(chaveVal)) return ''

    loading.value = true
    queryStore.clearResult()
    try {
      const input = {
        CompanyCNPJ: form.value.cnpj,
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
    allOptions,
    companyOptions,
    documentOptions,
    loadCompanies,
    filterCompanies,
    filterDocuments,
    runQuery,
  }
}
