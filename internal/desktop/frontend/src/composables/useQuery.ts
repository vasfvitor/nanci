import { ref } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useQueryStore } from '@/stores/query'

export function useQuery() {
  const queryStore = useQueryStore()
  const { form, result, type } = storeToRefs(queryStore)
  const loading = ref(false)
  const allOptions = ref<{ label: string; value: string }[]>([])
  const companyOptions = ref<{ label: string; value: string }[]>([])

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

  async function runQuery() {
    if (!form.value.cnpj || form.value.chave.length !== 50) return ''

    loading.value = true
    queryStore.clearResult()
    try {
      const input = {
        CNPJ: form.value.cnpj,
        ChaveAcesso: form.value.chave,
      }
      result.value =
        type.value === 'nfse'
          ? await desktopClient.queryNFSe(input)
          : await desktopClient.queryNFSeEvents(input)
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
    loadCompanies,
    filterCompanies,
    runQuery,
  }
}
