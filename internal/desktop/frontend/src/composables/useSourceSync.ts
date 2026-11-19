import { computed, type Ref } from 'vue'
import { useCompanySyncStore } from '@/stores/companySync'
import type { SyncSource } from '@/types/desktop'

// useSourceSync is the sync state a document page shows for the company's
// source: whether a pull (or an import, fetch or recovery holding the same
// marker) runs, whether it can be cancelled, and whether the source's reset
// runs. The markers live in the companySync store and in the page store's
// resettingCNPJ, so they survive leaving the page.
export function useSourceSync(source: SyncSource, cnpj: Ref<string>, resettingCNPJ: Ref<string>) {
  const syncStore = useCompanySyncStore()

  const isSyncing = computed(() => Boolean(cnpj.value) && syncStore.isSyncing(cnpj.value, source))
  const canCancelSync = computed(
    () => Boolean(cnpj.value) && syncStore.canCancelSync(cnpj.value, source)
  )
  const isCancelling = computed(
    () => Boolean(cnpj.value) && syncStore.isCancelling(cnpj.value, source)
  )
  const isResetting = computed(() => Boolean(cnpj.value) && resettingCNPJ.value === cnpj.value)

  // syncBusy is true without a company, while the company's source has an
  // operation in flight, or while its reset runs.
  function syncBusy(companyCNPJ: string) {
    return (
      !companyCNPJ ||
      syncStore.isSyncing(companyCNPJ, source) ||
      resettingCNPJ.value === companyCNPJ
    )
  }

  // cancelSync asks the backend to stop the pull. The pull stays busy until
  // the backend has stopped; then the page's sync refreshes it.
  async function cancelSync() {
    if (cnpj.value) await syncStore.cancelSync(cnpj.value, source)
  }

  return { isSyncing, canCancelSync, isCancelling, isResetting, syncBusy, cancelSync }
}
