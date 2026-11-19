import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useExportGuard } from '@/composables/useExportGuard'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useRowMap } from '@/composables/useRowMap'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useSefazStatus } from '@/composables/useSefazStatus'
import { useSourceSync } from '@/composables/useSourceSync'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import type { NFeExportFormat } from '@/types/desktop'
import { nfeNoteCount, nfePendingCount, nfeStateBadges, nfeStatusLine } from '@/utils/nfeDisplay'

export function useNFeDocuments() {
  const store = useNFeDocumentsStore()
  const workspace = useWorkspaceStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useNFeLoaders()
  const { cnpj, selectedCompany } = storeToRefs(workspace)
  const {
    filter,
    chaveFocus,
    rows,
    selected,
    rowsFor,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    activeTab,
    resettingCNPJ,
  } = storeToRefs(store)
  const pagination = useTablePagination('nfe')

  // filteredRows is what the grid shows: the search result narrowed by the
  // accent- and case-insensitive filterText.
  const { filteredRows } = useRowTextFilter({
    rows,
    filterText,
    fields: (row) => [
      row.ChaveAcesso,
      row.Numero,
      row.EmitenteCNPJ,
      row.EmitenteName,
      row.DestinatarioCNPJ,
      row.DestinatarioName,
    ],
    pagination,
  })

  const { onlyUnviewed, scopeRows, unviewedChaves, markViewed } = useMarkViewed({
    cnpj: () => store.listInput.CNPJ,
    filter,
    mark: (cnpj, chavesAcesso) => desktopClient.markNFeViewed(cnpj, chavesAcesso),
    rows,
    filteredRows,
    selected,
    setRows: (next) => store.setRows(next),
    markingViewed,
    listedOnlyUnread: () => Boolean(store.listInput.OnlyUnread),
  })

  const badgesByChave = useRowMap(rows, (row) => row.ChaveAcesso, nfeStateBadges)

  const pendingCount = computed(() => nfePendingCount(status.value))
  const noteCount = computed(() => nfeNoteCount(status.value))
  const statusLine = computed(() => (status.value ? nfeStatusLine(status.value) : ''))

  const { companyName, ambiente, idleWarning, syncBlockedUntil, blockedText } = useSefazStatus(
    status,
    selectedCompany
  )
  const { isSyncing, canCancelSync, isCancelling, isResetting, syncBusy, cancelSync } =
    useSourceSync('nfe', cnpj, resettingCNPJ)

  // syncNFe runs one distribution pull. The in-flight marker and the
  // cancellable mark live in the companySync store, so the button stays
  // busy, and the pull cancellable, after navigating away and back.
  async function syncNFe() {
    const companyCNPJ = cnpj.value
    if (syncBusy(companyCNPJ)) return null

    try {
      return await syncStore.runCancellableSync(companyCNPJ, 'nfe', () =>
        desktopClient.pullNFe(companyCNPJ)
      )
    } finally {
      await refresh(companyCNPJ)
    }
  }

  // importXML asks for XML files and stores them for the company without
  // asking SEFAZ. It holds the NF-e sync marker, like a pull, so the two
  // never overlap. null means it did not run or the dialog was cancelled.
  async function importXML() {
    const companyCNPJ = cnpj.value
    if (syncBusy(companyCNPJ)) return null

    const files = await desktopClient.selectXMLFiles()
    if (!files.length || syncBusy(companyCNPJ)) return null

    try {
      return await syncStore.runSync(companyCNPJ, 'nfe', () =>
        desktopClient.importNFeXML(companyCNPJ, files)
      )
    } finally {
      await refresh(companyCNPJ)
    }
  }

  // fetchByChave asks SEFAZ for one NF-e by its chave, spending one request
  // of the hourly budget. It holds the NF-e sync marker, like a pull.
  async function fetchByChave(chaveAcesso: string) {
    const companyCNPJ = cnpj.value
    if (syncBusy(companyCNPJ)) return null

    try {
      return await syncStore.runSync(companyCNPJ, 'nfe', () =>
        desktopClient.fetchNFeByChave(companyCNPJ, chaveAcesso)
      )
    } finally {
      await refresh(companyCNPJ)
    }
  }

  // resetNFe removes the company's NF-e and resets its NF-e sync. It never
  // runs alongside a pull, and its in-flight marker lives in the store so the
  // page stays busy after navigating away and back.
  async function resetNFe() {
    const companyCNPJ = cnpj.value
    if (!companyCNPJ || resettingCNPJ.value || syncStore.isSyncing(companyCNPJ, 'nfe')) return null

    resettingCNPJ.value = companyCNPJ
    try {
      return await desktopClient.resetNFe(companyCNPJ)
    } finally {
      resettingCNPJ.value = ''
      await refresh(companyCNPJ)
    }
  }

  // The exports read the company from listInput, like the search that
  // filled the grid.
  const { runExport } = useExportGuard(exporting, () => store.listInput.CNPJ)

  function exportXML(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportNFeXML({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  function exportDANFE(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportDANFE({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  // exportDocuments exports exactly the given chaves as a spreadsheet, a ZIP
  // of XMLs or a ZIP of DANFEs; incremental leaves out the ones exported
  // before and includeResumos adds the resumos, which have no complete XML.
  // Competence and Role are left empty: the grid may hold the result of an
  // earlier search, and an empty list would export everything.
  async function exportDocuments(
    format: NFeExportFormat,
    chavesAcesso: string[],
    choice: { incremental: boolean; includeResumos: boolean }
  ) {
    if (chavesAcesso.length === 0) return null
    return runExport((cnpj) =>
      desktopClient.exportNFeDocuments({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        Format: format,
        ChavesAcesso: chavesAcesso,
        IncludeResumos: choice.includeResumos,
        Incremental: choice.incremental,
      })
    )
  }

  // listEvents loads the events of one note of the company.
  function listEvents(chaveAcesso: string) {
    return desktopClient.listNFeEvents(cnpj.value, chaveAcesso)
  }

  return {
    listEvents,
    filter,
    chaveFocus,
    focusChave: (chave: string) => store.focusChave(chave),
    clearChaveFocus: () => store.clearChaveFocus(),
    selected,
    rowsFor,
    clearRows: () => store.clearRows(),
    loading,
    exporting,
    markingViewed,
    onlyUnviewed,
    status,
    activeTab,
    pagination,
    filterText,
    filteredRows,
    scopeRows,
    unviewedChaves,
    badgesByChave,
    companyName,
    ambiente,
    pendingCount,
    noteCount,
    statusLine,
    idleWarning,
    isSyncing,
    canCancelSync,
    isCancelling,
    isResetting,
    syncBlockedUntil,
    syncBusy,
    blockedText,
    search,
    loadStatus,
    syncNFe,
    cancelSync,
    importXML,
    fetchByChave,
    resetNFe,
    exportXML,
    exportDANFE,
    exportDocuments,
    markViewed,
  }
}
