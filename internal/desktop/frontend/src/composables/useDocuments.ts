import { ref } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useDocumentsStore } from '@/stores/documents'
import type { CompanySummary, ExportFormat } from '@/types/desktop'

export function useDocuments() {
  const documentsStore = useDocumentsStore()
  const { filter, documents } = storeToRefs(documentsStore)
  const companyOptions = ref<{ label: string; value: string }[]>([])
  const loading = ref(false)

  async function loadCompanies() {
    const companies = await desktopClient.listCompanies()
    companyOptions.value = companies.map(companyOption)
    return companies
  }

  async function search() {
    if (!filter.value.CNPJ) return []
    loading.value = true
    try {
      const rows = await desktopClient.listDocuments({
        CNPJ: filter.value.CNPJ,
        Competence: filter.value.Competence || '',
        Direction: filter.value.Direction || '',
      })
      documentsStore.setDocuments(rows)
      return rows
    } finally {
      loading.value = false
    }
  }

  async function exportDocuments(format: ExportFormat) {
    const outDir = await desktopClient.selectExportDirectory()
    if (!outDir) return null
    return desktopClient.exportDocuments({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
      Format: format,
      OutDir: outDir,
    })
  }

  async function exportDANFSe(chaveAcesso: string) {
    const outDir = await desktopClient.selectExportDirectory()
    if (!outDir) return null
    return desktopClient.exportDANFSe({
      CNPJ: filter.value.CNPJ,
      ChaveAcesso: chaveAcesso,
      OutDir: outDir,
    })
  }

  async function exportDANFSeZIP() {
    const outDir = await desktopClient.selectExportDirectory()
    if (!outDir) return null
    return desktopClient.exportDANFSeZIP({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
      Format: 'zip',
      OutDir: outDir,
    })
  }

  async function loadEvents(documentID: string) {
    return desktopClient.listEventsForDocument(documentID)
  }

  return {
    filter,
    documents,
    companyOptions,
    loading,
    loadCompanies,
    search,
    exportDocuments,
    exportDANFSe,
    exportDANFSeZIP,
    loadEvents,
  }
}

function companyOption(company: CompanySummary) {
  return {
    label: `${company.Name} (${company.CNPJ})`,
    value: company.CNPJ,
  }
}
