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
  const isResetting = computed(() => Boolean(cnpj.value) && resettingCNPJ.value === cnpj.value)

  const { syncBlockedUntil, blockedText } = useSefazBlock(status)

  // syncCTe runs one distribution pull. The in-flight marker lives in the
  // companySync store so the button stays busy after navigating away and back.
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
      return await syncStore.runSync(companyCNPJ, 'cte', () => desktopClient.pullCTe(companyCNPJ))
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

  // exportZIP exports exactly the given chaves; incremental leaves out the
  // ones exported before. Competence and Role are left empty: the grid may
  // hold the result of an earlier search, and an empty list would export
  // everything.
  async function exportZIP(chavesAcesso: string[], choice: { incremental: boolean }) {
    if (chavesAcesso.length === 0) return null
    return runExport((cnpj) =>
      desktopClient.exportCTeZIP({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
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
    isResetting,
    syncBlockedUntil,
    blockedText,
    search,
    loadStatus,
    syncCTe,
    previewReset,
    resetCTe,
    exportXML,
    exportZIP,
    markViewed,
  }
}
