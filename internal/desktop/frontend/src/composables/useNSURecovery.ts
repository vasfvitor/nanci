import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useQuasar } from 'quasar'
import { useCTeLoaders } from '@/composables/useCTeLoaders'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useNotify } from '@/composables/useNotify'
import { useSefazBlock } from '@/composables/useSefazBlock'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import type { RecoverNSUResult } from '@/types/desktop'
import {
  missingNSUsText,
  recoveryAllowance,
  recoveryConfirmText,
  recoverySummary,
} from '@/utils/nsuRecovery'

export type NSURecoverySource = 'nfe' | 'cte'

// useNSURecovery holds the "Recuperar NSUs" action of the NF-e or CT-e page:
// it asks SEFAZ, one consNSU at a time, for the NSUs the distribution
// skipped. The run holds the source's sync marker in the companySync store,
// like a pull, so it stays visible and cancellable after leaving the page.
export function useNSURecovery(source: NSURecoverySource) {
  const $q = useQuasar()
  const { notifySuccess, notifyWarning, notifySyncError } = useNotify()
  const syncStore = useCompanySyncStore()
  const { cnpj } = storeToRefs(useWorkspaceStore())
  const store = source === 'nfe' ? useNFeDocumentsStore() : useCTeDocumentsStore()
  const { refresh } = source === 'nfe' ? useNFeLoaders() : useCTeLoaders()
  const recoverNSU = (companyCNPJ: string) =>
    source === 'nfe'
      ? desktopClient.recoverNFeNSU(companyCNPJ)
      : desktopClient.recoverCTeNSU(companyCNPJ)

  const status = computed(() => store.status)
  const missing = computed(() => status.value?.NSUFaltantes ?? 0)
  const allowance = computed(() => recoveryAllowance(status.value))
  const notice = computed(() => (missing.value > 0 ? missingNSUsText(missing.value) : ''))

  // The hour of wait after the queue caught up only holds back distNSU; a
  // 656 or the spent budget hold back consNSU too.
  const { syncBlockedUntil } = useSefazBlock(status)
  const blocked = computed(
    () => Boolean(syncBlockedUntil.value) && status.value?.BlockedReason !== 'caught_up'
  )

  function busy(companyCNPJ: string) {
    return (
      !companyCNPJ ||
      syncStore.isSyncing(companyCNPJ, source) ||
      store.resettingCNPJ === companyCNPJ
    )
  }

  const disabledReason = computed(() => {
    if (busy(cnpj.value)) return 'Aguarde a operação em andamento terminar.'
    if (blocked.value) return 'Consultas bloqueadas no momento. Aguarde o horário indicado.'
    if (allowance.value === 0) {
      return 'Sem consultas sobrando nesta hora; 2 ficam para a sincronização.'
    }
    return ''
  })

  // recover runs one recovery; null means it did not run.
  async function recover(): Promise<RecoverNSUResult | null> {
    const companyCNPJ = cnpj.value
    if (busy(companyCNPJ)) return null
    try {
      const result = await syncStore.runCancellableSync(companyCNPJ, source, () =>
        recoverNSU(companyCNPJ)
      )
      const notify = result.StopReason === 'consumo_indevido' ? notifyWarning : notifySuccess
      notify(recoverySummary(result))
      return result
    } catch (error) {
      notifySyncError('Erro ao recuperar NSUs', error)
      return null
    } finally {
      await refresh(companyCNPJ)
    }
  }

  function confirmRecover() {
    if (missing.value === 0 || disabledReason.value) return
    $q.dialog({
      title: 'Recuperar NSUs faltantes',
      message: recoveryConfirmText(
        missing.value,
        allowance.value,
        status.value?.RequestBudget ?? 0
      ),
      cancel: true,
      persistent: true,
      ok: { label: 'Recuperar', color: 'primary' },
    }).onOk(() => {
      void recover()
    })
  }

  return { missing, notice, disabledReason, recover, confirmRecover }
}
