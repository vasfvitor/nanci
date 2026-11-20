import { useQuasar } from 'quasar'
import CTeDesacordoDialog from '@/components/CTeDesacordoDialog.vue'
import { useCTeDesacordo } from '@/composables/useCTeDesacordo'
import { useNotify } from '@/composables/useNotify'
import { wailsErrorCode } from '@/platform/wails/client'
import type { CTeDesacordoResult, CTeRow } from '@/types/desktop'

// useCTeDesacordoActions holds the prestação em desacordo flow of the CT-e
// page: the dialog that plans it and the notifications of the answer. The
// requests and the in-flight marker go through useCTeDesacordo.
export function useCTeDesacordoActions() {
  const $q = useQuasar()
  const { notifyError, notifySuccess, notifyInfo, notifyWarning } = useNotify()
  const desacordo = useCTeDesacordo()

  function openDesacordo(row: CTeRow) {
    $q.dialog({
      component: CTeDesacordoDialog,
      componentProps: {
        row,
        load: (observacao: string) => desacordo.planDesacordo(row.ChaveAcesso, observacao),
      },
    }).onOk((observacao: string) => {
      void sendDesacordo(row.ChaveAcesso, observacao)
    })
  }

  async function sendDesacordo(chaveAcesso: string, observacao: string) {
    try {
      const result = await desacordo.registerDesacordo(chaveAcesso, observacao)
      if (!result) {
        notifyWarning('Este CT-e já tem um envio em andamento.')
        return
      }
      notifyDesacordoResult(result)
    } catch (error) {
      if (wailsErrorCode(error) === 'canceled') {
        notifyWarning('Envio da prestação em desacordo cancelado.')
      } else {
        notifyError('Erro ao registrar a prestação em desacordo', error)
      }
    }
  }

  function notifyDesacordoResult(result: CTeDesacordoResult) {
    const detail = [result.CStat, result.XMotivo].filter(Boolean).join(' - ')
    switch (result.Status) {
      case 'registrada':
        notifySuccess(`Prestação em desacordo registrada. Protocolo ${result.Protocolo || '—'}.`)
        return
      case 'ja_registrada':
        notifyInfo('A prestação em desacordo já estava registrada na SEFAZ.')
        return
      case 'rejeitada':
        notifyError('Prestação em desacordo rejeitada pela SEFAZ', detail)
        return
      default:
        notifyWarning(`Prestação em desacordo não enviada. ${detail}`.trim())
    }
  }

  return { openDesacordo, isSending: desacordo.isSending }
}
