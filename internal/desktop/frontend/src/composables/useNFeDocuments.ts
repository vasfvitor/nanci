import { computed, getCurrentScope, onScopeDispose, ref, shallowRef, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { companyOption } from '@/composables/useCompanies'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useTablePagination } from '@/composables/useTablePagination'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { CompanySummary } from '@/types/desktop'
import { parseDate } from '@/utils/formatters'

export type NFeExportZIPOptions = {
  includeResumos?: boolean
  incremental?: boolean
}

export function useNFeDocuments() {
  const store = useNFeDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { search, loadStatus, refresh } = useNFeLoaders()
  const { filter, rows, selected, loading, exporting, status, activeTab, resettingCNPJ } =
    storeToRefs(store)
  const companyOptions = ref<{ label: string; value: string }[]>([])
  const pagination = useTablePagination()

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

  async function loadCompanies(): Promise<CompanySummary[]> {
    const companies = await desktopClient.listCompanies()
    companyOptions.value = companies.map(companyOption)
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

  async function exportXML(chaveAcesso: string) {
    if (exporting.value) return null
    exporting.value = true
    try {
      return await desktopClient.exportNFeXML({
        CNPJ: filter.value.CNPJ,
        ChaveAcesso: chaveAcesso,
      })
    } finally {
      exporting.value = false
    }
  }

  // exportZIP exports exactly the given chaves, the rows the user sees or
  // selected. Competence and Role are left empty: the grid may hold the
  // result of an earlier search, and an empty list would export everything.
  async function exportZIP(chavesAcesso: string[], options: NFeExportZIPOptions = {}) {
    if (exporting.value || chavesAcesso.length === 0) return null
    exporting.value = true
    try {
      return await desktopClient.exportNFeZIP({
        CNPJ: store.listInput.CNPJ,
        Competence: '',
        Role: '',
        ChavesAcesso: chavesAcesso,
        IncludeResumos: options.includeResumos ?? false,
        Incremental: options.incremental ?? false,
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
    companyOptions,
    isSyncing,
    isResetting,
    syncBlockedUntil,
    loadCompanies,
    search,
    loadStatus,
    syncNFe,
    resetNFe,
    exportXML,
    exportZIP,
  }
}
