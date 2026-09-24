import { storeToRefs } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'

// useNFeLoaders loads NF-e notes, status and pendências into the
// nfeDocuments store. A result that arrives after the user picked another
// company is dropped.
export function useNFeLoaders() {
  const store = useNFeDocumentsStore()
  const { loading, status, pending, pendingLoading } = storeToRefs(store)

  const isSelected = (cnpj: string) => store.filter.CNPJ === cnpj

  async function search() {
    const input = store.listInput
    if (!input.CNPJ) return []
    loading.value = true
    try {
      const result = await desktopClient.listNFe(input)
      if (isSelected(input.CNPJ)) store.setRows(result)
      return result
    } finally {
      loading.value = false
    }
  }

  async function loadStatus(cnpj: string = store.filter.CNPJ) {
    if (!cnpj) {
      status.value = null
      return null
    }
    const result = await desktopClient.statusNFe(cnpj)
    if (isSelected(cnpj)) status.value = result
    return result
  }

  async function loadPending(cnpj: string = store.filter.CNPJ) {
    if (!cnpj) {
      pending.value = []
      return []
    }
    pendingLoading.value = true
    try {
      const rows = await desktopClient.listNFePendingManifestacoes(cnpj)
      if (isSelected(cnpj)) pending.value = rows
      return rows
    } finally {
      pendingLoading.value = false
    }
  }

  // reloadNote fetches one listed note with the current filters and patches
  // it in the list; a note that no longer matches them leaves the list, as a
  // full search would do.
  async function reloadNote(chaveAcesso: string) {
    const input = { ...store.listInput, ChavesAcesso: [chaveAcesso] }
    const result = await desktopClient.listNFe(input)
    if (isSelected(input.CNPJ)) store.patchRow(chaveAcesso, result[0] ?? null)
  }

  // refresh reloads notes, status and pendências after work that already
  // happened: a sync, a reset or an event registered at SEFAZ. It also runs
  // after a failed pull, because the status then carries the block reason.
  // Its own failures must not hide the result of that work.
  async function refresh(cnpj: string) {
    if (!isSelected(cnpj)) return
    await Promise.allSettled([search(), loadStatus(cnpj), loadPending(cnpj)])
  }

  // refreshNote is refresh after an event on one note: only that note is
  // reloaded when it is listed. An unlisted note may now match the filters,
  // so then the whole list is searched again.
  async function refreshNote(cnpj: string, chaveAcesso: string) {
    if (!isSelected(cnpj)) return
    const listed = store.rows.some((row) => row.ChaveAcesso === chaveAcesso)
    await Promise.allSettled([
      listed ? reloadNote(chaveAcesso) : search(),
      loadStatus(cnpj),
      loadPending(cnpj),
    ])
  }

  return { search, loadStatus, loadPending, refresh, refreshNote }
}
