import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompanyFilter } from '@/composables/useCompanyFilter'
import { useDocumentLoaders } from '@/composables/useDocumentLoaders'
import { useExportGuard } from '@/composables/useExportGuard'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useRowMap } from '@/composables/useRowMap'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useDocumentsStore } from '@/stores/documents'
import type { ExportFormat } from '@/types/desktop'
import { nfseAmbiente, nfseStateBadges, nfseStatusLine } from '@/utils/nfseDisplay'

// useDocuments holds the NFS-e page: the list, its filters and the actions
// on the listed NFS-e. The state that outlives the page is in the documents
// store; the NFS-e sync is the one the Empresas page runs, so both screens
// share its busy state through the companySync store. The NFS-e has no
// status call: its sync state comes with the company list, which a sync or
// a reset reloads once, in refresh.
export function useDocuments() {
  const store = useDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { isSelected, search } = useDocumentLoaders(store, (input) =>
    desktopClient.listDocuments(input)
  )
  const {
    filter,
    documents,
    selected,
    filterText,
    loading,
    exporting,
    markingViewed,
    resettingCNPJ,
  } = storeToRefs(store)
  const { companyOptions, selectedCompany, loadCompanies } = useCompanyFilter(filter)
  const pagination = useTablePagination('nfse')

  // filteredRows is what the grid shows: the search result narrowed by the
  // accent- and case-insensitive filterText.
  const { filteredRows } = useRowTextFilter({
    rows: documents,
    filterText,
    fields: (row) => [
      row.ChaveAcesso,
      row.NFSeNumber,
      row.PrestadorCNPJ,
      row.PrestadorName,
      row.TomadorCNPJ,
      row.TomadorName,
      row.Status,
      row.ServiceDescription,
    ],
    pagination,
  })

  const { onlyUnviewed, scopeRows, unviewedChaves, markViewed } = useMarkViewed({
    filter,
    mark: (cnpj, chavesAcesso) => desktopClient.markDocumentsViewed(cnpj, chavesAcesso),
    rows: documents,
    filteredRows,
    selected,
    setRows: (rows) => store.setRows(rows),
    markingViewed,
  })

  const badgesByChave = useRowMap(documents, (row) => row.ChaveAcesso, nfseStateBadges)

  const ambiente = computed(() =>
    selectedCompany.value ? nfseAmbiente(selectedCompany.value.Environment) : null
  )
  const statusLine = computed(() =>
    selectedCompany.value ? nfseStatusLine(selectedCompany.value) : ''
  )

  const isSyncing = computed(
    () => Boolean(filter.value.CNPJ) && syncStore.isSyncing(filter.value.CNPJ, 'nfse')
  )
  const isResetting = computed(
    () => Boolean(filter.value.CNPJ) && resettingCNPJ.value === filter.value.CNPJ
  )

  // refresh reloads the list and the company list, whose sync fields feed
  // the status line, after a sync or a reset. Its own failures are not
  // reported, so they never hide the result of the sync.
  async function refresh(companyCNPJ: string) {
    if (!isSelected(companyCNPJ)) return
    await Promise.allSettled([search(), loadCompanies()])
  }

  // syncNFSe runs one ADN pull for the selected company.
  async function syncNFSe() {
    const companyCNPJ = filter.value.CNPJ
    if (
      !companyCNPJ ||
      syncStore.isSyncing(companyCNPJ, 'nfse') ||
      resettingCNPJ.value === companyCNPJ
    ) {
      return null
    }
    try {
      return await syncStore.runSync(companyCNPJ, 'nfse', () =>
        desktopClient.pull({ CNPJ: companyCNPJ, Mode: '' })
      )
    } finally {
      await refresh(companyCNPJ)
    }
  }

  // resetSync restarts the company's NFS-e sync from NSU 0. It moves only
  // the cursor: the documents stay.
  async function resetSync() {
    const companyCNPJ = filter.value.CNPJ
    if (!companyCNPJ || resettingCNPJ.value || syncStore.isSyncing(companyCNPJ, 'nfse'))
      return false
    resettingCNPJ.value = companyCNPJ
    try {
      await desktopClient.resetSyncState({ CompanyCNPJ: companyCNPJ })
      return true
    } finally {
      resettingCNPJ.value = ''
      await refresh(companyCNPJ)
    }
  }

  // The exports read the company from listInput, like the search that
  // filled the grid.
  const { runExport } = useExportGuard(exporting, () => store.listInput.CNPJ)

  function exportXML(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportXML({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  function exportDANFSe(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportDANFSe({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  // exportDocuments exports exactly the given NFS-e as a spreadsheet or a ZIP
  // of XMLs, and exportDANFSeZIP as a ZIP of DANFSes; incremental leaves out
  // the ones exported before. Competence and Direction are left empty: the
  // grid may hold the result of an earlier search, and an empty list would
  // export everything.
  async function exportDocuments(
    format: ExportFormat,
    chavesAcesso: string[],
    choice: { incremental: boolean }
  ) {
    if (chavesAcesso.length === 0) return null
    return runExport((cnpj) =>
      desktopClient.exportDocuments({
        CNPJ: cnpj,
        Competence: '',
        Direction: '',
        Format: format,
        Incremental: choice.incremental,
        ChavesAcesso: chavesAcesso,
      })
    )
  }

  async function exportDANFSeZIP(chavesAcesso: string[], choice: { incremental: boolean }) {
    if (chavesAcesso.length === 0) return null
    return runExport((cnpj) =>
      desktopClient.exportDANFSeZIP({
        CNPJ: cnpj,
        Competence: '',
        Direction: '',
        Format: 'zip',
        Incremental: choice.incremental,
        ChavesAcesso: chavesAcesso,
      })
    )
  }

  return {
    filter,
    documents,
    selected,
    loading,
    exporting,
    markingViewed,
    onlyUnviewed,
    pagination,
    filterText,
    filteredRows,
    scopeRows,
    unviewedChaves,
    badgesByChave,
    companyOptions,
    selectedCompany,
    ambiente,
    statusLine,
    isSyncing,
    isResetting,
    loadCompanies,
    search,
    syncNFSe,
    resetSync,
    exportXML,
    exportDANFSe,
    exportDocuments,
    exportDANFSeZIP,
    markViewed,
  }
}
