import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { companyOption } from '@/composables/useCompanies'
import { isNFeChaveFilter, useCTeLoaders } from '@/composables/useCTeLoaders'
import { useSefazBlock } from '@/composables/useSefazBlock'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import type { CompanySummary } from '@/types/desktop'
import { normalizeText } from '@/utils/formatters'
import { cteDocumentCount, cteStatusLine } from '@/utils/cteDisplay'

export function useCTeDocuments() {
  const store = useCTeDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useCTeLoaders()
  const { filter, rows, loading, exporting, incremental, status, resettingCNPJ } =
    storeToRefs(store)
  const companyOptions = ref<{ label: string; value: string }[]>([])
  const pagination = useTablePagination()
  const filterText = ref('')

  // searchIndex normalizes the searchable fields once per result set, not on
  // every keystroke.
  const searchIndex = computed(() =>
    rows.value.map((row) => ({
      row,
      fields: [
        row.ChaveAcesso,
        row.Numero,
        row.EmitenteCNPJ,
        row.EmitenteName,
        row.TomadorCNPJ,
        row.TomadorName,
      ].map(normalizeText),
    }))
  )

  // filteredRows is what the grid shows: the search result narrowed by the
  // accent- and case-insensitive filterText.
  const filteredRows = computed(() => {
    const query = normalizeText(filterText.value)
    if (!query) return rows.value
    return searchIndex.value
      .filter(({ fields }) => fields.some((field) => field.includes(query)))
      .map(({ row }) => row)
  })

  watch(filterText, () => {
    pagination.value.page = 1
  })

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

  // loadCompanies lists the companies and keeps a known one selected,
  // falling back to the first.
  async function loadCompanies(): Promise<CompanySummary[]> {
    const companies = await desktopClient.listCompanies()
    companyOptions.value = companies.map(companyOption)
    if (!companyOptions.value.some((option) => option.value === filter.value.CNPJ)) {
      filter.value.CNPJ = companyOptions.value[0]?.value ?? ''
    }
    return companies
  }

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

  // exportZIP exports exactly the given chaves, by default every row the
  // grid shows; incremental leaves out the ones exported before. Competence
  // and Role are left empty: the grid may hold the result of an earlier
  // search, and an empty list would export everything.
  async function exportZIP(chavesAcesso: string[] = filteredRows.value.map((row) => row.ChaveAcesso)) {
    const cnpj = store.listInput.CNPJ
    if (!cnpj || exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportCTeZIP({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        ChavesAcesso: chavesAcesso,
        Incremental: incremental.value,
      })
    } finally {
      exporting.value = false
    }
  }

  return {
    filter,
    rows,
    loading,
    exporting,
    incremental,
    status,
    pagination,
    filterText,
    filteredRows,
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
  }
}
