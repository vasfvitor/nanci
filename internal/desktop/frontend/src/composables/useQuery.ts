import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useQueryStore } from '@/stores/query'
import { useWorkspaceStore } from '@/stores/workspace'
import { latestOnly } from '@/utils/latestOnly'

export function useQuery() {
  const queryStore = useQueryStore()
  const { form, result, type, loading } = storeToRefs(queryStore)
  // The query authenticates as the workspace company.
  const workspace = useWorkspaceStore()
  const { cnpj, selectedCompany } = storeToRefs(workspace)

  const allDocumentOptions = ref<{ label: string; value: string; description?: string }[]>([])
  const documentOptions = ref<{ label: string; value: string; description?: string }[]>([])

  const documentGate = latestOnly()

  // The chave suggestions are the NFS-e of the workspace company. A fetch
  // that ends after the company changed again is dropped.
  watch(
    cnpj,
    async (newCnpj) => {
      const isLatest = documentGate.begin()

      if (!newCnpj) {
        allDocumentOptions.value = []
        documentOptions.value = []
        return
      }
      try {
        const docs = await desktopClient.listDocuments({ CNPJ: newCnpj, Competence: '', Direction: '', OnlyUnread: false })
        if (!isLatest()) return

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
    cnpj,
    selectedCompany,
    documentOptions,
    filterDocuments,
    runQuery,
  }
}
