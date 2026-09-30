import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompanyFilter } from '@/composables/useCompanyFilter'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useSefazBlock } from '@/composables/useSefazBlock'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeRow } from '@/types/desktop'
import { nfeNoteCount, nfePendingCount, nfeStatusLine } from '@/utils/nfeDisplay'
import { nfeRowActions } from '@/utils/nfeManifestacao'

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
  const cnpj = computed({
    get: () => filter.value.CNPJ,
    set: (value: string) => {
      filter.value.CNPJ = value
    },
  })
  const { companyOptions, loadCompanies } = useCompanyFilter(cnpj)
  const pagination = useTablePagination('nfe')
  const { onlyUnviewed, markViewed } = useMarkViewed({
    filter,
    mark: (cnpj, chavesAcesso) =>
      desktopClient.markNFeViewed({
        CNPJ: cnpj,
        Competence: '',
        Situacao: '',
        Completeness: '',
        Manifestacao: '',
        Role: '',
        EmitenteCNPJ: '',
        ChavesAcesso: chavesAcesso,
      }),
    rows,
    selected,
    setRows: (next) => store.setRows(next),
    search,
    markingViewed,
  })

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
    onChange: () => {
      pagination.value.page = 1
    },
  })

  // scopeRows are the NF-e "Marcar vistos" and "Exportar" act on: the
  // selection, or else every row the grid shows.
  const scopeRows = computed(() => (selected.value.length > 0 ? selected.value : filteredRows.value))
  const unviewedChaves = computed(() =>
    scopeRows.value.filter((row) => !row.ViewedAt).map((row) => row.ChaveAcesso)
  )

  // actionsByChave holds the row menu state of every row the grid shows,
  // computed once per result set rather than on each render of a row.
  const actionsByChave = computed(
    () => new Map(filteredRows.value.map((row) => [row.ChaveAcesso, nfeRowActions(row)]))
  )

  function rowActions(row: NFeRow) {
    return actionsByChave.value.get(row.ChaveAcesso) ?? nfeRowActions(row)
  }

  const companyName = computed(() => {
    if (status.value?.CompanyName) return status.value.CompanyName
    const option = companyOptions.value.find((item) => item.value === filter.value.CNPJ)
    return option?.label ?? ''
  })
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

    syncStore.startSync(cnpj, 'nfe')
    try {
      return await desktopClient.pullNFe(cnpj)
    } finally {
      syncStore.finishSync(cnpj, 'nfe')
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
  async function exportXML(chaveAcesso: string) {
    const cnpj = store.listInput.CNPJ
    if (!cnpj || exporting.value) return null
    exporting.value = true
    try {
      return await desktopClient.exportNFeXML({ CNPJ: cnpj, ChaveAcesso: chaveAcesso })
    } finally {
      exporting.value = false
    }
  }

  // exportZIP exports exactly the given chaves; incremental leaves out the
  // ones exported before and includeResumos adds the resumos, which have no
  // complete XML. Competence and Role are left empty: the grid may hold the
  // result of an earlier search, and an empty list would export everything.
  async function exportZIP(
    chavesAcesso: string[],
    choice: { incremental: boolean; includeResumos: boolean }
  ) {
    const cnpj = store.listInput.CNPJ
    if (!cnpj || exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportNFeZIP({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        ChavesAcesso: chavesAcesso,
        IncludeResumos: choice.includeResumos,
        Incremental: choice.incremental,
      })
    } finally {
      exporting.value = false
    }
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
    companyOptions,
    companyName,
    pendingCount,
    noteCount,
    statusLine,
    isSyncing,
    isResetting,
    syncBlockedUntil,
    blockedText,
    rowActions,
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
