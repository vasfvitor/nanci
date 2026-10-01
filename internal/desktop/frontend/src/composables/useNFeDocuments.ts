import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompanyFilter } from '@/composables/useCompanyFilter'
import { useExportGuard } from '@/composables/useExportGuard'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useRowMap } from '@/composables/useRowMap'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useSefazBlock } from '@/composables/useSefazBlock'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { nfeNoteCount, nfePendingCount, nfeStateBadges, nfeStatusLine } from '@/utils/nfeDisplay'
import { sefazAmbiente } from '@/utils/sefazDisplay'

export function useNFeDocuments() {
  const store = useNFeDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useNFeLoaders()
  const {
    filter,
    rows,
    selected,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    activeTab,
    resettingCNPJ,
  } = storeToRefs(store)
  const { companyOptions, selectedCompany, loadCompanies } = useCompanyFilter(filter)
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
    filter,
    mark: (cnpj, chavesAcesso) => desktopClient.markNFeViewed(cnpj, chavesAcesso),
    rows,
    filteredRows,
    selected,
    setRows: (next) => store.setRows(next),
    markingViewed,
  })

  const badgesByChave = useRowMap(rows, (row) => row.ChaveAcesso, nfeStateBadges)

  const companyName = computed(() => status.value?.CompanyName || selectedCompany.value?.Name || '')
  const ambiente = computed(() => (status.value ? sefazAmbiente(status.value.TpAmb) : null))
  const pendingCount = computed(() => nfePendingCount(status.value))
  const noteCount = computed(() => nfeNoteCount(status.value))
  const statusLine = computed(() => (status.value ? nfeStatusLine(status.value) : ''))

  const isSyncing = computed(
    () => Boolean(filter.value.CNPJ) && syncStore.isSyncing(filter.value.CNPJ, 'nfe')
  )
  const isResetting = computed(
    () => Boolean(filter.value.CNPJ) && resettingCNPJ.value === filter.value.CNPJ
  )

  const { syncBlockedUntil, blockedText } = useSefazBlock(status)

  // syncNFe runs one distribution pull. The in-flight marker lives in the
  // companySync store so the button stays busy after navigating away and back.
  async function syncNFe() {
    const cnpj = filter.value.CNPJ
    if (!cnpj || syncStore.isSyncing(cnpj, 'nfe') || resettingCNPJ.value === cnpj) return null

    try {
      return await syncStore.runSync(cnpj, 'nfe', () => desktopClient.pullNFe(cnpj))
    } finally {
      await refresh(cnpj)
    }
  }

  // resetNFe removes the company's NF-e and resets its NF-e sync. It never
  // runs alongside a pull, and its in-flight marker lives in the store so the
  // page stays busy after navigating away and back.
  async function resetNFe() {
    const cnpj = filter.value.CNPJ
    if (!cnpj || resettingCNPJ.value || syncStore.isSyncing(cnpj, 'nfe')) return null

    resettingCNPJ.value = cnpj
    try {
      return await desktopClient.resetNFe(cnpj)
    } finally {
      resettingCNPJ.value = ''
      await refresh(cnpj)
    }
  }

  // The exports read the company from listInput, like the search that
  // filled the grid.
  const { runExport } = useExportGuard(exporting, () => store.listInput.CNPJ)

  function exportXML(chaveAcesso: string) {
    return runExport((cnpj) => desktopClient.exportNFeXML({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
  }

  // exportZIP exports exactly the given chaves; incremental leaves out the
  // ones exported before and includeResumos adds the resumos, which have no
  // complete XML. Competence and Role are left empty: the grid may hold the
  // result of an earlier search, and an empty list would export everything.
  async function exportZIP(
    chavesAcesso: string[],
    choice: { incremental: boolean; includeResumos: boolean }
  ) {
    if (chavesAcesso.length === 0) return null
    return runExport((cnpj) =>
      desktopClient.exportNFeZIP({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        ChavesAcesso: chavesAcesso,
        IncludeResumos: choice.includeResumos,
        Incremental: choice.incremental,
      })
    )
  }

  return {
    filter,
    rows,
    selected,
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
    companyOptions,
    companyName,
    ambiente,
    pendingCount,
    noteCount,
    statusLine,
    isSyncing,
    isResetting,
    syncBlockedUntil,
    blockedText,
    loadCompanies,
    search,
    loadStatus,
    syncNFe,
    resetNFe,
    exportXML,
    exportZIP,
    markViewed,
  }
}
