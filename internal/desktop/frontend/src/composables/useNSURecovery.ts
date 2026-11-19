import { computed, getCurrentScope, onScopeDispose, shallowRef, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useQuasar } from 'quasar'
import { useCTeLoaders } from '@/composables/useCTeLoaders'
import { useNFeLoaders } from '@/composables/useNFeLoaders'
import { useNotify } from '@/composables/useNotify'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import type { RecoverNSUResult } from '@/types/desktop'
import { formatTime, parseDate } from '@/utils/formatters'
import {
  discardConfirmText,
  discardedNSUsText,
  missingNSUsText,
  recoveryConfirmText,
  recoverySummary,
} from '@/utils/nsuRecovery'

export type NSURecoverySource = 'nfe' | 'cte'

export type NSURecoveryOptions = {
  // syncBusy tells whether a pull, import, fetch or reset of the company's
  // source runs; it is the page composable's own check.
  syncBusy: (cnpj: string) => boolean
}

// useNSURecovery holds the "Recuperar NSUs" action of the NF-e or CT-e page:
// it asks SEFAZ, one consNSU at a time, for the NSUs the distribution
// skipped. The run holds the source's sync marker in the companySync store,
// like a pull, so it stays visible and cancellable after leaving the page.
// How many NSUs a run asks for and when it may start come from the status
// (RecoveryAllowance, RecoveryReadyAt), which the backend computes.
export function useNSURecovery(source: NSURecoverySource, options: NSURecoveryOptions) {
  const $q = useQuasar()
  const { notifySuccess, notifyWarning, notifySyncError } = useNotify()
  const syncStore = useCompanySyncStore()
  const { cnpj } = storeToRefs(useWorkspaceStore())
  const store = source === 'nfe' ? useNFeDocumentsStore() : useCTeDocumentsStore()
  const { refresh, loadStatus } = source === 'nfe' ? useNFeLoaders() : useCTeLoaders()
  const recoverNSU = (companyCNPJ: string) =>
    source === 'nfe'
      ? desktopClient.recoverNFeNSU(companyCNPJ)
      : desktopClient.recoverCTeNSU(companyCNPJ)
  const discardNSU = (companyCNPJ: string) =>
    source === 'nfe'
      ? desktopClient.discardNFeNSU(companyCNPJ)
      : desktopClient.discardCTeNSU(companyCNPJ)

  const status = computed(() => store.status)
  const missing = computed(() => status.value?.NSUFaltantes ?? 0)
  const allowance = computed(() => status.value?.RecoveryAllowance ?? 0)
  const notice = computed(() => (missing.value > 0 ? missingNSUsText(missing.value) : ''))

  // now ticks when RecoveryReadyAt passes, and the status is loaded again for
  // the new allowance, so the button comes back without a page reload.
  const now = shallowRef(Date.now())
  let readyTimer: ReturnType<typeof setTimeout> | undefined
  watch(
    () => status.value?.RecoveryReadyAt,
    (value) => {
      now.value = Date.now()
      clearTimeout(readyTimer)
      const at = parseDate(value)
      if (at && at.getTime() > now.value) {
        readyTimer = setTimeout(
          () => {
            now.value = Date.now()
            if (cnpj.value) void loadStatus(cnpj.value).catch(() => {})
          },
          at.getTime() - now.value + 1000
        )
      }
    },
    { immediate: true }
  )
  if (getCurrentScope()) onScopeDispose(() => clearTimeout(readyTimer))

  const readyAt = computed(() => {
    const at = parseDate(status.value?.RecoveryReadyAt)
    return at && at.getTime() > now.value ? at : null
  })

  const disabledReason = computed(() => {
    if (options.syncBusy(cnpj.value)) return 'Aguarde a operação em andamento terminar.'
    if (readyAt.value)
      return `Sem consultas para recuperar agora; volta às ${formatTime(readyAt.value)}.`
    if (allowance.value === 0) return 'Sem consultas sobrando nesta hora.'
    return ''
  })

  // recover runs one recovery; null means it did not run.
  async function recover(): Promise<RecoverNSUResult | null> {
    const companyCNPJ = cnpj.value
    if (options.syncBusy(companyCNPJ)) return null
    try {
      const result = await syncStore.runCancellableSync(companyCNPJ, source, () =>
        recoverNSU(companyCNPJ)
      )
      const stoppedShort =
        result.StopReason === 'consumo_indevido' || result.StopReason === 'canceled'
      const notify = stoppedShort ? notifyWarning : notifySuccess
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

  // discard closes the gaps so recovery never asks for them; null means it
  // did not run. It holds the sync marker like a recovery, though it asks
  // SEFAZ nothing.
  async function discard(): Promise<number | null> {
    const companyCNPJ = cnpj.value
    if (options.syncBusy(companyCNPJ)) return null
    try {
      const discarded = await syncStore.runSync(companyCNPJ, source, () => discardNSU(companyCNPJ))
      notifySuccess(discardedNSUsText(discarded))
      return discarded
    } catch (error) {
      notifySyncError('Erro ao descartar NSUs', error)
      return null
    } finally {
      await refresh(companyCNPJ)
    }
  }

  function confirmDiscard() {
    if (missing.value === 0 || options.syncBusy(cnpj.value)) return
    $q.dialog({
      title: 'Descartar NSUs faltantes',
      message: discardConfirmText(missing.value),
      cancel: true,
      persistent: true,
      ok: { label: 'Descartar', color: 'negative' },
    }).onOk(() => {
      void discard()
    })
  }

  return { missing, notice, disabledReason, recover, confirmRecover, discard, confirmDiscard }
}
