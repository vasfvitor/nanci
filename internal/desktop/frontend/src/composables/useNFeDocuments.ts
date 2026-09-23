import { computed, getCurrentScope, onScopeDispose, ref, shallowRef, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { usePreferencesStore } from '@/stores/preferences'
import type { CompanySummary, ISODateValue } from '@/types/desktop'

export type NFeExportZIPOptions = {
  includeResumos?: boolean
  incremental?: boolean
}

function toDate(value: ISODateValue): Date | null {
  if (!value) return null
  const parsed = value instanceof Date ? value : new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

export function useNFeDocuments() {
  const store = useNFeDocumentsStore()
  const syncStore = useCompanySyncStore()
  const { filter, rows, selected, loading, exporting, status, activeTab } = storeToRefs(store)
  const companyOptions = ref<{ label: string; value: string }[]>([])

  const preferencesStore = usePreferencesStore()
  const { rowsPerPage } = storeToRefs(preferencesStore)

  const pagination = ref({
    sortBy: 'issueDate',
    descending: true,
    page: 1,
    rowsPerPage: rowsPerPage.value,
  })

  watch(
    () => pagination.value.rowsPerPage,
    (newVal) => {
      rowsPerPage.value = newVal
    }
  )

  const isSyncing = computed(
    () => Boolean(filter.value.CNPJ) && syncStore.isSyncing(filter.value.CNPJ, 'nfe')
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
      const until = toDate(value)
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
    const until = toDate(status.value?.NextAllowedAt)
    return until && until.getTime() > now.value ? until : null
  })

  async function loadCompanies(): Promise<CompanySummary[]> {
    const companies = await desktopClient.listCompanies()
    companyOptions.value = companies.map(companyOption)
    return companies
  }

  async function search() {
    const input = store.listInput
    if (!input.CNPJ) return []
    loading.value = true
    try {
      const result = await desktopClient.listNFe(input)
      if (store.filter.CNPJ === input.CNPJ) {
        store.setRows(result)
      }
      return result
    } finally {
      loading.value = false
    }
  }

  async function loadStatus() {
    const cnpj = filter.value.CNPJ
    if (!cnpj) {
      store.setStatus(null)
      return null
    }
    const result = await desktopClient.statusNFe(cnpj)
    if (store.filter.CNPJ === cnpj) {
      store.setStatus(result)
    }
    return result
  }

  // syncNFe runs one distribution pull. The in-flight marker lives in the
  // companySync store so the button stays busy after navigating away and back.
  async function syncNFe() {
    const cnpj = filter.value.CNPJ
    if (!cnpj || syncStore.isSyncing(cnpj, 'nfe')) return null

    syncStore.startSync(cnpj, 'nfe')
    try {
      return await desktopClient.pullNFe(cnpj)
    } finally {
      syncStore.finishSync(cnpj, 'nfe')
      await refreshAfterSync(cnpj)
    }
  }

  // refreshAfterSync also runs after a failed pull, because the status then
  // carries the block reason. Its own errors must not hide the pull result.
  async function refreshAfterSync(cnpj: string) {
    if (store.filter.CNPJ !== cnpj) return
    await Promise.allSettled([search(), loadStatus()])
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

  async function loadEvents(chaveAcesso: string, cnpj: string = filter.value.CNPJ) {
    return desktopClient.listNFeEvents(cnpj, chaveAcesso)
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
    syncBlockedUntil,
    loadCompanies,
    search,
    loadStatus,
    syncNFe,
    exportXML,
    exportZIP,
    loadEvents,
  }
}

function companyOption(company: CompanySummary) {
  return {
    label: `${company.Name} (${company.CNPJ})`,
    value: company.CNPJ,
  }
}
