import { computed, getCurrentScope, onScopeDispose, ref, shallowRef, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { companyOption } from '@/composables/useCompanies'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { CompanySummary, NFeRow } from '@/types/desktop'
import { formatTime, normalizeText, parseDate } from '@/utils/formatters'
import { nfeNoteCount, nfePendingCount, nfeStatusLine } from '@/utils/nfeDisplay'
import { blockedMessage } from '@/utils/sefazDisplay'
import { nfeRowActions } from '@/utils/nfeManifestacao'

export function useNFeDocuments() {
  const store = useNFeDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useNFeLoaders()
  const { filter, rows, selected, loading, exporting, status, activeTab, resettingCNPJ } =
    storeToRefs(store)
  const companyOptions = ref<{ label: string; value: string }[]>([])
  const pagination = useTablePagination()
  const filterText = ref('')

  // searchIndex normalizes the searchable fields once per result set, not on
  // every keystroke.
  const searchIndex = computed(() =>
    rows.value.map((row) => ({
      row,
      fields: [row.ChaveAcesso, row.Numero, row.EmitenteCNPJ, row.EmitenteName].map(normalizeText),
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

  // now ticks when the SEFAZ block ends, so syncBlockedUntil clears itself
  // without polling.
  const now = shallowRef(Date.now())
  let unblockTimer: ReturnType<typeof setTimeout> | undefined

  watch(
    () => status.value?.NextAllowedAt,
    (value) => {
      now.value = Date.now()
      clearTimeout(unblockTimer)
      const until = parseDate(value)
      if (until && until.getTime() > now.value) {
        unblockTimer = setTimeout(() => {
          now.value = Date.now()
        }, until.getTime() - now.value + 1000)
      }
    },
    { immediate: true }
  )

  if (getCurrentScope()) {
    onScopeDispose(() => clearTimeout(unblockTimer))
  }

  const syncBlockedUntil = computed(() => {
    const until = parseDate(status.value?.NextAllowedAt)
    return until && until.getTime() > now.value ? until : null
  })

  const blockedText = computed(() => {
    if (!syncBlockedUntil.value || !status.value) return ''
    return blockedMessage(status.value, formatTime(syncBlockedUntil.value))
  })

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

  // exportChaves are the chaves a ZIP export takes by default: the selected
  // rows, or else every row the grid shows.
  const exportChaves = computed(() =>
    (selected.value.length > 0 ? selected.value : filteredRows.value).map((row) => row.ChaveAcesso)
  )

  // exportZIP exports exactly the given chaves, by default exportChaves.
  // Competence and Role are left empty: the grid may hold the result of an
  // earlier search, and an empty list would export everything.
  async function exportZIP(chavesAcesso: string[] = exportChaves.value) {
    const cnpj = store.listInput.CNPJ
    if (!cnpj || exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportNFeZIP({
        CNPJ: cnpj,
        Competence: '',
        Role: '',
        ChavesAcesso: chavesAcesso,
        IncludeResumos: false,
        Incremental: false,
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
    status,
    activeTab,
    pagination,
    filterText,
    filteredRows,
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
  }
}
