import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useDocumentsStore } from '@/stores/documents'
import { usePreferencesStore } from '@/stores/preferences'
import type { CompanySummary, ExportFormat } from '@/types/desktop'

export function useDocuments() {
  const documentsStore = useDocumentsStore()
  const { filter, documents, loading, exporting } = storeToRefs(documentsStore)
  const companyOptions = ref<{ label: string; value: string }[]>([])

  const preferencesStore = usePreferencesStore()
  const { rowsPerPage } = storeToRefs(preferencesStore)

  const pagination = ref({
    sortBy: 'issueDate',
    descending: true,
    page: 1,
    rowsPerPage: rowsPerPage.value
  })

  watch(
    () => pagination.value.rowsPerPage,
    (newVal) => {
      rowsPerPage.value = newVal
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
        OnlyUnread: filter.value.OnlyUnread || false,
      })
      documentsStore.setDocuments(rows)
      return rows
    } finally {
      loading.value = false
    }
  }

  async function markDocumentsViewed() {
    if (!filter.value.CNPJ) return 0
    loading.value = true
    try {
      const count = await desktopClient.markDocumentsViewed({
        CNPJ: filter.value.CNPJ,
        Competence: filter.value.Competence || '',
        Direction: filter.value.Direction || '',
        OnlyUnread: filter.value.OnlyUnread || false,
      })
      // Reload documents after marking as viewed to update the state
      await search()
      return count
    } finally {
      loading.value = false
    }
  }

  async function exportDocuments(format: ExportFormat, incremental: boolean = false, outPath: string = '', chavesAcesso: string[] = []) {
    if (exporting.value) return
    exporting.value = true
    try {
      return await desktopClient.exportDocuments({
        CNPJ: filter.value.CNPJ,
        Competence: filter.value.Competence || '',
        Direction: filter.value.Direction || '',
        Format: format,
        OutPath: outPath,
        Incremental: incremental,
        ChavesAcesso: chavesAcesso,
      })
    } finally {
      exporting.value = false
    }
  }

  async function exportDANFSe(chaveAcesso: string) {
    if (exporting.value) return
    exporting.value = true
    try {
      return await desktopClient.exportDANFSe({
        CNPJ: filter.value.CNPJ,
        ChaveAcesso: chaveAcesso,
      })
    } finally {
      exporting.value = false
    }
  }

  async function exportXML(chaveAcesso: string) {
    if (exporting.value) return
    exporting.value = true
    try {
      return await desktopClient.exportXML({
        CNPJ: filter.value.CNPJ,
        ChaveAcesso: chaveAcesso,
      })
    } finally {
      exporting.value = false
    }
  }

  async function exportDANFSeZIP(incremental: boolean = false, outPath: string = '', chavesAcesso: string[] = []) {
    if (exporting.value) return
    exporting.value = true
    try {
      return await desktopClient.exportDANFSeZIP({
        CNPJ: filter.value.CNPJ,
        Competence: filter.value.Competence || '',
        Direction: filter.value.Direction || '',
        Format: 'zip',
        OutPath: outPath,
        Incremental: incremental,
        ChavesAcesso: chavesAcesso,
      })
    } finally {
      exporting.value = false
    }
  }

  async function countPendingExports(format: ExportFormat | 'danfse-zip') {
    if (!filter.value.CNPJ) return 0
    return desktopClient.countPendingExports({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
      Format: format as ExportFormat,
      OutPath: '',
      Incremental: true,
      ChavesAcesso: [],
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
    markDocumentsViewed,
    countPendingExports,
  }
}

function companyOption(company: CompanySummary) {
  return {
    label: `${company.Name} (${company.CNPJ})`,
    value: company.CNPJ,
  }
}
