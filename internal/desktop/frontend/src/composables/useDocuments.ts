import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useDocumentsStore } from '@/stores/documents'
import type { CompanySummary, ExportFormat } from '@/types/desktop'

export function useDocuments() {
  const documentsStore = useDocumentsStore()
  const { filter, documents, loading, exporting } = storeToRefs(documentsStore)
  const companyOptions = ref<{ label: string; value: string }[]>([])

  const savedRowsPerPage = Number(localStorage.getItem('nanci:documents:rowsPerPage')) || 25
  const pagination = ref({
    sortBy: 'issueDate',
    descending: true,
    page: 1,
    rowsPerPage: savedRowsPerPage
  })

  watch(
    () => pagination.value.rowsPerPage,
    (newVal) => {
      localStorage.setItem('nanci:documents:rowsPerPage', String(newVal))
    }
  )

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
    return desktopClient.exportDocuments({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
      Format: format,
    })
  }

  async function exportDANFSe(chaveAcesso: string) {
    return desktopClient.exportDANFSe({
      CNPJ: filter.value.CNPJ,
      ChaveAcesso: chaveAcesso,
    })
  }

  async function exportXML(chaveAcesso: string) {
    return desktopClient.exportXML({
      CNPJ: filter.value.CNPJ,
      ChaveAcesso: chaveAcesso,
    })
  }

  async function exportDANFSeZIP() {
    return desktopClient.exportDANFSeZIP({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
      Format: 'zip',
    })
  }

  async function loadEvents(documentID: string) {
    return desktopClient.listEventsForDocument(documentID)
  }

  return {
    filter,
    documents,
    pagination,
    companyOptions,
    loading,
    exporting,
    loadCompanies,
    search,
    exportDocuments,
    exportDANFSe,
    exportXML,
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
