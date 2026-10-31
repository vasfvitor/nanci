import type { Ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useQuasar } from 'quasar'
import NFeCienciaConfirmDialog from '@/components/NFeCienciaConfirmDialog.vue'
import NFeEventResultsDialog from '@/components/NFeEventResultsDialog.vue'
import NFeManifestacaoDialog from '@/components/NFeManifestacaoDialog.vue'
import { useNFeManifestacao } from '@/composables/useNFeManifestacao'
import { useNotify } from '@/composables/useNotify'
import { wailsErrorCode } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type {
  NFeConclusiveTipo,
  NFeEventBatchResult,
  NFeEventResult,
  NFeRow,
} from '@/types/desktop'
import { countOutcomes, isProblemOutcome, noEligibleCienciaMessage } from '@/utils/nfeManifestacao'

export type NFeManifestacaoActionsOptions = {
  cnpj: Ref<string>
  companyName: Ref<string>
  // tpAmb is the ambiente of the status, shown in the dialogs.
  tpAmb: Ref<string>
  // syncNFe is the "Sincronizar agora" action of the ciência notice.
  syncNFe: () => Promise<void>
}

// useNFeManifestacaoActions holds the ciência and manifestação flows of the
// NF-e page: the confirmation dialogs, the notifications and the results
// dialog. The requests and their in-flight markers go through
// useNFeManifestacao.
export function useNFeManifestacaoActions(options: NFeManifestacaoActionsOptions) {
  const $q = useQuasar()
  const { notifyError, notifySuccess, notifyInfo, notifyWarning } = useNotify()
  const manifestacao = useNFeManifestacao()
  const { selected } = storeToRefs(useNFeDocumentsStore())

  function registerSelectedCiencia() {
    if (manifestacao.eligibleSelection.value.length === 0) return
    void startCiencia(selected.value.map((row) => row.ChaveAcesso))
  }

  // startCiencia asks the backend which notes can receive ciência, then asks
  // the user to confirm exactly those notes.
  async function startCiencia(chavesAcesso: string[]) {
    let plan
    try {
      plan = await manifestacao.planCiencia(chavesAcesso)
    } catch (error) {
      notifyError('Erro ao preparar a ciência', error)
      return
    }
    if (!plan) return

    const noEligible = noEligibleCienciaMessage(plan)
    if (noEligible) {
      notifyWarning(noEligible)
      return
    }

    $q.dialog({
      component: NFeCienciaConfirmDialog,
      componentProps: {
        companyName: options.companyName.value,
        cnpj: options.cnpj.value,
        tpAmb: options.tpAmb.value,
        plan,
      },
    }).onOk((eligible: string[]) => {
      void sendCiencia(eligible)
    })
  }

  async function sendCiencia(chavesAcesso: string[]) {
    try {
      const result = await manifestacao.registerCiencia(chavesAcesso)
      if (!result) {
        notifyWarning('Já existe um envio de ciência em andamento.')
        return
      }
      notifyCienciaResult(result)
    } catch (error) {
      if (wailsErrorCode(error) === 'canceled') {
        notifyWarning('Envio de ciência cancelado.')
      } else {
        notifyError('Erro ao registrar ciência', error)
      }
    }
  }

  function notifyCienciaResult(result: NFeEventBatchResult) {
    const counts = countOutcomes(result.Results)
    const hasProblems = result.Results.some(isProblemOutcome) || Boolean(result.Interrupted)
    const notify = hasProblems ? notifyWarning : notifySuccess
    notify(
      `Ciência: ${counts.registrada} registradas, ${counts.ja_registrada} já registradas, ${counts.rejeitada} rejeitadas, ${counts.nao_enviada} não enviadas.`,
      {
        caption: 'O XML completo chega na próxima sincronização.',
        timeout: 10000,
        actions: [
          {
            label: 'Sincronizar agora',
            handler: () => {
              void options.syncNFe()
            },
          },
        ],
      }
    )

    if (hasProblems) {
      $q.dialog({ component: NFeEventResultsDialog, componentProps: { result } })
    }
  }

  function openManifestacao(row: NFeRow) {
    $q.dialog({
      component: NFeManifestacaoDialog,
      componentProps: {
        note: row,
        tpAmb: options.tpAmb.value,
      },
    }).onOk((payload: { tipo: NFeConclusiveTipo; justificativa: string }) => {
      void sendManifestacao(row.ChaveAcesso, payload.tipo, payload.justificativa)
    })
  }

  async function sendManifestacao(
    chaveAcesso: string,
    tipo: NFeConclusiveTipo,
    justificativa: string
  ) {
    try {
      const result = await manifestacao.registerManifestacao(chaveAcesso, tipo, justificativa)
      if (!result) {
        notifyWarning('Esta nota já tem um envio em andamento.')
        return
      }
      notifyManifestacaoResult(result)
    } catch (error) {
      if (wailsErrorCode(error) === 'canceled') {
        notifyWarning('Envio da manifestação cancelado.')
      } else {
        notifyError('Erro ao registrar manifestação', error)
      }
    }
  }

  function notifyManifestacaoResult(result: NFeEventResult) {
    const detail = [result.CStat, result.XMotivo].filter(Boolean).join(' - ')
    switch (result.Status) {
      case 'registrada':
        notifySuccess(`Manifestação registrada. Protocolo ${result.Protocolo || '—'}.`)
        return
      case 'ja_registrada':
        notifyInfo('A manifestação já estava registrada na SEFAZ.')
        return
      case 'rejeitada':
        notifyError('Manifestação rejeitada pela SEFAZ', detail)
        return
      default:
        notifyWarning(`Manifestação não enviada. ${detail}`.trim())
    }
  }

  return { registerSelectedCiencia, startCiencia, openManifestacao }
}
