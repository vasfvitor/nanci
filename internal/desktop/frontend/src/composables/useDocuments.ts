import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompanyFilter } from '@/composables/useCompanyFilter'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { useNFSeLoaders } from '@/composables/useNFSeLoaders'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useDocumentsStore } from '@/stores/documents'
import type { ExportFormat } from '@/types/desktop'
import { nfseAmbiente, nfseStatusLine } from '@/utils/nfseDisplay'

// useDocuments holds the NFS-e page: the list, its filters and the actions
// on the listed NFS-e. The state that outlives the page is in the documents
// store; the NFS-e sync is the one the Empresas page runs, so both screens
// share its busy state through the companySync store. A sync or a reset
// reloads the company list once, in refresh.
export function useDocuments() {
  const store = useDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { isSelected, search } = useNFSeLoaders()
  const { filter, documents, selected, filterText, loading, exporting, markingViewed, resettingCNPJ } =
    storeToRefs(store)
  const cnpj = computed({
    get: () => filter.value.CNPJ,
    set: (value: string) => {
      filter.value.CNPJ = value
    },
  })
  const { companyOptions, selectedCompany, loadCompanies } = useCompanyFilter(cnpj)
  const pagination = useTablePagination('nfse')
  const { onlyUnviewed, markViewed } = useMarkViewed({
    filter,
    mark: (companyCNPJ, chavesAcesso) =>
      desktopClient.markDocumentsViewed({
        CNPJ: companyCNPJ,
        Competence: '',
        Direction: '',
        OnlyUnread: false,
        ChavesAcesso: chavesAcesso,
      }),
    rows: documents,
    selected,
    setRows: (rows) => store.setRows(rows),
    search,
    markingViewed,
  })

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
    onChange: () => {
      pagination.value.page = 1
    },
  })

  // scopeRows are the NFS-e "Marcar vistos" and "Exportar" act on: the
  // selection, or else every row the grid shows.
  const scopeRows = computed(() => (selected.value.length > 0 ? selected.value : filteredRows.value))
  const unviewedChaves = computed(() =>
    scopeRows.value.filter((row) => !row.ViewedAt).map((row) => row.ChaveAcesso)
  )

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
  // the status line, after a sync or a reset. Its own failures must not hide
  // the result of that work.
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
    syncStore.startSync(companyCNPJ, 'nfse')
    try {
      return await desktopClient.pull({ CNPJ: companyCNPJ, Mode: '' })
    } finally {
      syncStore.finishSync(companyCNPJ, 'nfse')
      await refresh(companyCNPJ)
    }
  }

  // resetSync restarts the company's NFS-e sync from NSU 0. It moves only
  // the cursor: the documents stay.
  async function resetSync() {
    const companyCNPJ = filter.value.CNPJ
    if (!companyCNPJ || resettingCNPJ.value || syncStore.isSyncing(companyCNPJ, 'nfse')) return false
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
  async function exportXML(chaveAcesso: string) {
    const companyCNPJ = store.listInput.CNPJ
    if (!companyCNPJ || exporting.value) return null
    exporting.value = true
    try {
      return await desktopClient.exportXML({ CNPJ: companyCNPJ, ChaveAcesso: chaveAcesso })
    } finally {
      exporting.value = false
    }
  }

  async function exportDANFSe(chaveAcesso: string) {
    const companyCNPJ = store.listInput.CNPJ
    if (!companyCNPJ || exporting.value) return null
    exporting.value = true
    try {
      return await desktopClient.exportDANFSe({ CNPJ: companyCNPJ, ChaveAcesso: chaveAcesso })
    } finally {
      exporting.value = false
    }
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
    const companyCNPJ = store.listInput.CNPJ
    if (!companyCNPJ || exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportDocuments({
        CNPJ: companyCNPJ,
        Competence: '',
        Direction: '',
        Format: format,
        Incremental: choice.incremental,
        ChavesAcesso: chavesAcesso,
      })
    } finally {
      exporting.value = false
    }
  }

  async function exportDANFSeZIP(chavesAcesso: string[], choice: { incremental: boolean }) {
    const companyCNPJ = store.listInput.CNPJ
    if (!companyCNPJ || exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportDANFSeZIP({
        CNPJ: companyCNPJ,
        Competence: '',
        Direction: '',
        Format: 'zip',
        Incremental: choice.incremental,
        ChavesAcesso: chavesAcesso,
      })
    } finally {
      exporting.value = false
    }
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
