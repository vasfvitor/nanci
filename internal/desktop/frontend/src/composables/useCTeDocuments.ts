import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useCTeLoaders } from '@/composables/useCTeLoaders'
import { useExportGuard } from '@/composables/useExportGuard'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useRowMap } from '@/composables/useRowMap'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useSefazBlock } from '@/composables/useSefazBlock'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CTeExportFormat } from '@/types/desktop'
import { cteDocumentCount, cteStateBadges, cteStatusLine } from '@/utils/cteDisplay'
import { idleWarning as idleWarningText, sefazAmbiente } from '@/utils/sefazDisplay'

export function useCTeDocuments() {
  const store = useCTeDocumentsStore()
  const workspace = useWorkspaceStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useCTeLoaders()
  const { cnpj, selectedCompany } = storeToRefs(workspace)
  const {
    filter,
    listError,
    rows,
    selected,
    rowsFor,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    resettingCNPJ,
  } = storeToRefs(store)
  const pagination = useTablePagination('cte')

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
      row.TomadorCNPJ,
      row.TomadorName,
    ],
    pagination,
  })

  const { onlyUnviewed, scopeRows, unviewedChaves, markViewed } = useMarkViewed({
    cnpj: () => store.listInput.CNPJ,
    filter,
    mark: (cnpj, chavesAcesso) => desktopClient.markCTeViewed(cnpj, chavesAcesso),
    rows,
    filteredRows,
    selected,
    setRows: (next) => store.setRows(next),
    markingViewed,
  })

  const badgesByChave = useRowMap(rows, (row) => row.ChaveAcesso, cteStateBadges)

  const companyName = computed(() => status.value?.CompanyName || selectedCompany.value?.Name || '')
  const ambiente = computed(() => (status.value ? sefazAmbiente(status.value.TpAmb) : null))
  const documentCount = computed(() => cteDocumentCount(status.value))
  const statusLine = computed(() => (status.value ? cteStatusLine(status.value) : ''))
  const idleWarning = computed(() => idleWarningText(status.value))

  const isSyncing = computed(() => Boolean(cnpj.value) && syncStore.isSyncing(cnpj.value, 'cte'))
  const canCancelSync = computed(
    () => Boolean(cnpj.value) && syncStore.canCancelSync(cnpj.value, 'cte')
  )
  const isCancelling = computed(
    () => Boolean(cnpj.value) && syncStore.isCancelling(cnpj.value, 'cte')
  )
  const isResetting = computed(() => Boolean(cnpj.value) && resettingCNPJ.value === cnpj.value)

  const { syncBlockedUntil, blockedText } = useSefazBlock(status)

  // syncCTe runs one distribution pull. The in-flight marker and the
  // cancellable mark live in the companySync store, so the button stays
  // busy, and the pull cancellable, after navigating away and back.
  async function syncCTe() {
    const companyCNPJ = cnpj.value
    if (
      !companyCNPJ ||
      syncStore.isSyncing(companyCNPJ, 'cte') ||
      resettingCNPJ.value === companyCNPJ
    ) {
      return null
    }

    try {
      return await syncStore.runCancellableSync(companyCNPJ, 'cte', () =>
        desktopClient.pullCTe(companyCNPJ)
      )
    } finally {
      await refresh(companyCNPJ)
    }
  }

  // cancelSync asks the backend to stop the pull. The pull stays busy until
  // the backend has stopped; then syncCTe refreshes the page.
  async function cancelSync() {
    if (cnpj.value) await syncStore.cancelSync(cnpj.value, 'cte')
  }

  // syncBusy is true without a company, while a pull, import or fetch of the
  // company's CT-e runs, or while its reset runs.
  function syncBusy(companyCNPJ: string) {
    return (
      !companyCNPJ || syncStore.isSyncing(companyCNPJ, 'cte') || resettingCNPJ.value === companyCNPJ
    )
  }

  // importXML asks for XML files and stores them for the company without
  // asking SEFAZ. It holds the CT-e sync marker, like a pull, so the two
  // never overlap. null means it did not run or the dialog was cancelled.
  async function importXML() {
    const companyCNPJ = cnpj.value
    if (syncBusy(companyCNPJ)) return null

    const files = await desktopClient.selectXMLFiles()
    if (!files.length || syncBusy(companyCNPJ)) return null

    try {
      return await syncStore.runSync(companyCNPJ, 'cte', () =>
        desktopClient.importCTeXML(companyCNPJ, files)
      )
    } finally {
      await refresh(companyCNPJ)
    }
  }

  // previewReset counts what resetCTe would remove, for the confirmation.
  async function previewReset() {
    const companyCNPJ = cnpj.value
    if (!companyCNPJ || resettingCNPJ.value || syncStore.isSyncing(companyCNPJ, 'cte')) return null
    return desktopClient.previewResetCTe(companyCNPJ)
  }

  // resetCTe removes the company's CT-e and resets its CT-e sync. It never
  // runs alongside a pull, and its in-flight marker lives in the store so the
  // page stays busy after navigating away and back.
  async function resetCTe() {
    const companyCNPJ = cnpj.value
    if (!companyCNPJ || resettingCNPJ.value || syncStore.isSyncing(companyCNPJ, 'cte')) return null

    resettingCNPJ.value = companyCNPJ
    try {
      return await desktopClient.resetCTe(companyCNPJ)
    } finally {
      resettingCNPJ.value = ''
      await refresh(companyCNPJ)
    }
  }

  // The exports read the company from listInput, like the search that
  // filled the grid.
  const { runExport } = useExportGuard(exporting, () => store.listInput.CNPJ)

  function exportXML(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportCTeXML({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  function exportDACTE(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportDACTE({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  // exportDocuments exports exactly the given chaves as a spreadsheet, a ZIP
  // of XMLs or a ZIP of DACTEs; incremental leaves out the ones exported
  // before. Competence and Role are left empty: the grid may hold the result
  // of an earlier search, and an empty list would export everything.
  async function exportDocuments(
    format: CTeExportFormat,
    chavesAcesso: string[],
    choice: { incremental: boolean }
  ) {
    if (chavesAcesso.length === 0) return null
    return runExport((cnpj) =>
      desktopClient.exportCTeDocuments({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        Format: format,
        ChavesAcesso: chavesAcesso,
        Incremental: choice.incremental,
      })
    )
  }

  return {
    filter,
    listError,
    rows,
    selected,
    rowsFor,
    clearRows: () => store.clearRows(),
    loading,
    exporting,
    markingViewed,
    onlyUnviewed,
    status,
    pagination,
    filterText,
    filteredRows,
    scopeRows,
    unviewedChaves,
    badgesByChave,
    companyName,
    ambiente,
    documentCount,
    statusLine,
    idleWarning,
    isSyncing,
    canCancelSync,
    isCancelling,
    isResetting,
    syncBlockedUntil,
    blockedText,
    search,
    loadStatus,
    syncCTe,
    cancelSync,
    importXML,
    previewReset,
    resetCTe,
    exportXML,
    exportDACTE,
    exportDocuments,
    markViewed,
  }
}
