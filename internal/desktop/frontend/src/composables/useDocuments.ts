import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useDocumentLoaders } from '@/composables/useDocumentLoaders'
import { useExportGuard } from '@/composables/useExportGuard'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useRowMap } from '@/composables/useRowMap'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useDocumentsStore } from '@/stores/documents'
import { useWorkspaceStore } from '@/stores/workspace'
import type { ExportFormat } from '@/types/desktop'
import { nfseAmbiente, nfseStateBadges, nfseStatusLine } from '@/utils/nfseDisplay'

// useDocuments holds the NFS-e page: the list, its filters and the actions
// on the listed NFS-e. The state that outlives the page is in the documents
// store, and the sync busy state in the companySync store, so a remounted
// page sees a sync in flight. The company and the competência come from the
// workspace. The NFS-e has no status call: its sync state comes with the
// workspace company list, which a sync or a reset reloads once, in refresh.
export function useDocuments() {
  const store = useDocumentsStore()
  const workspace = useWorkspaceStore()
  const syncStore = useCompanySyncStore()
  const { isSelected, search } = useDocumentLoaders(store, (input) =>
    desktopClient.listDocuments(input)
  )
  const { cnpj, competence, selectedCompany } = storeToRefs(workspace)
  const {
    filter,
    rows,
    selected,
    rowsFor,
    filterText,
    loading,
    exporting,
    markingViewed,
    resettingCNPJ,
  } = storeToRefs(store)
  const pagination = useTablePagination('nfse')

  // filteredRows is what the grid shows: the search result narrowed by the
  // accent- and case-insensitive filterText.
  const { filteredRows } = useRowTextFilter({
    rows,
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
    cnpj: () => store.listInput.CNPJ,
    filter,
    mark: (cnpj, chavesAcesso) => desktopClient.markDocumentsViewed(cnpj, chavesAcesso),
    rows,
    filteredRows,
    selected,
    setRows: (rows) => store.setRows(rows),
    markingViewed,
  })

  const badgesByChave = useRowMap(rows, (row) => row.ChaveAcesso, nfseStateBadges)

  const ambiente = computed(() =>
    selectedCompany.value ? nfseAmbiente(selectedCompany.value.Environment) : null
  )
  const statusLine = computed(() =>
    selectedCompany.value ? nfseStatusLine(selectedCompany.value) : ''
  )

  const isSyncing = computed(() => Boolean(cnpj.value) && syncStore.isSyncing(cnpj.value, 'nfse'))
  const isResetting = computed(() => Boolean(cnpj.value) && resettingCNPJ.value === cnpj.value)

  // refresh reloads the company list, whose sync fields feed the status line
  // and the Empresas page, after a sync or a reset, and the NFS-e list when
  // the company is still the selected one. The company list reloads either
  // way, so a sync of another company does not leave its fields old. Its own
  // failures are not reported, so they never hide the result of the sync.
  async function refresh(companyCNPJ: string) {
    await Promise.allSettled([
      workspace.loadCompanies(),
      isSelected(companyCNPJ) ? search() : Promise.resolve([]),
    ])
  }

  // syncNFSe runs one ADN pull for the selected company.
  async function syncNFSe() {
    const companyCNPJ = cnpj.value
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
    const companyCNPJ = cnpj.value
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
    rows,
    selected,
    rowsFor,
    clearRows: () => store.clearRows(),
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
    cnpj,
    competence,
    selectedCompany,
    ambiente,
    statusLine,
    isSyncing,
    isResetting,
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
