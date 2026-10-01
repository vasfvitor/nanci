import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompanyFilter } from '@/composables/useCompanyFilter'
import { useMarkViewed } from '@/composables/useMarkViewed'
import { isNFeChaveFilter, useCTeLoaders } from '@/composables/useCTeLoaders'
import { useRowTextFilter } from '@/composables/useRowTextFilter'
import { useSefazBlock } from '@/composables/useSefazBlock'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import { cteDocumentCount, cteStatusLine } from '@/utils/cteDisplay'

export function useCTeDocuments() {
  const store = useCTeDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useCTeLoaders()
  const {
    filter,
    rows,
    selected,
    filterText,
    loading,
    exporting,
    markingViewed,
    status,
    resettingCNPJ,
  } = storeToRefs(store)
  const cnpj = computed({
    get: () => filter.value.CNPJ,
    set: (value: string) => {
      filter.value.CNPJ = value
    },
  })
  const { companyOptions, loadCompanies } = useCompanyFilter(cnpj)
  const pagination = useTablePagination('cte')
  const { onlyUnviewed, markViewed } = useMarkViewed({
    filter,
    mark: (cnpj, chavesAcesso) => desktopClient.markCTeViewed(cnpj, chavesAcesso),
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
      row.TomadorCNPJ,
      row.TomadorName,
    ],
    onChange: () => {
      pagination.value.page = 1
    },
  })

  // scopeRows are the CT-e "Marcar vistos" and "Exportar" act on: the
  // selection, or else every row the grid shows.
  const scopeRows = computed(() => (selected.value.length > 0 ? selected.value : filteredRows.value))
  const unviewedChaves = computed(() =>
    scopeRows.value.filter((row) => !row.ViewedAt).map((row) => row.ChaveAcesso)
  )

  const companyName = computed(() => {
    if (status.value?.CompanyName) return status.value.CompanyName
    const option = companyOptions.value.find((item) => item.value === filter.value.CNPJ)
    return option?.label ?? ''
  })
  const documentCount = computed(() => cteDocumentCount(status.value))
  const statusLine = computed(() => (status.value ? cteStatusLine(status.value) : ''))

  // nfeChaveError explains why the NF-e key filter cannot be sent, or is ''.
  const nfeChaveError = computed(() =>
    isNFeChaveFilter(store.listInput.NFeChave) ? '' : 'A chave de NF-e tem 44 caracteres'
  )

  const isSyncing = computed(
    () => Boolean(filter.value.CNPJ) && syncStore.isSyncing(filter.value.CNPJ, 'cte')
  )
  const isResetting = computed(
    () => Boolean(filter.value.CNPJ) && resettingCNPJ.value === filter.value.CNPJ
  )

  const { syncBlockedUntil, blockedText } = useSefazBlock(status)

  // syncCTe runs one distribution pull. The in-flight marker lives in the
  // companySync store so the button stays busy after navigating away and back.
  async function syncCTe() {
    const cnpj = filter.value.CNPJ
    if (!cnpj || syncStore.isSyncing(cnpj, 'cte') || resettingCNPJ.value === cnpj) return null

    syncStore.startSync(cnpj, 'cte')
    try {
      return await desktopClient.pullCTe(cnpj)
    } finally {
      syncStore.finishSync(cnpj, 'cte')
      await refresh(cnpj)
    }
  }

  // previewReset counts what resetCTe would remove, for the confirmation.
  async function previewReset() {
    const cnpj = filter.value.CNPJ
    if (!cnpj || resettingCNPJ.value || syncStore.isSyncing(cnpj, 'cte')) return null
    return desktopClient.previewResetCTe(cnpj)
  }

  // resetCTe removes the company's CT-e and resets its CT-e sync. It never
  // runs alongside a pull, and its in-flight marker lives in the store so the
  // page stays busy after navigating away and back.
  async function resetCTe() {
    const cnpj = filter.value.CNPJ
    if (!cnpj || resettingCNPJ.value || syncStore.isSyncing(cnpj, 'cte')) return null

    resettingCNPJ.value = cnpj
    try {
      return await desktopClient.resetCTe(cnpj)
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
      return await desktopClient.exportCTeXML({ CNPJ: cnpj, ChaveAcesso: chaveAcesso })
    } finally {
      exporting.value = false
    }
  }

  // exportZIP exports exactly the given chaves; incremental leaves out the
  // ones exported before. Competence and Role are left empty: the grid may
  // hold the result of an earlier search, and an empty list would export
  // everything.
  async function exportZIP(chavesAcesso: string[], choice: { incremental: boolean }) {
    const cnpj = store.listInput.CNPJ
    if (!cnpj || exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportCTeZIP({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        ChavesAcesso: chavesAcesso,
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
    pagination,
    filterText,
    filteredRows,
    scopeRows,
    unviewedChaves,
    companyOptions,
    companyName,
    documentCount,
    statusLine,
    nfeChaveError,
    isSyncing,
    isResetting,
    syncBlockedUntil,
    blockedText,
    loadCompanies,
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
