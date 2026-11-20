import { useQuasar } from 'quasar'
import CTeDesacordoDialog from '@/components/CTeDesacordoDialog.vue'
import { useCTeDesacordo } from '@/composables/useCTeDesacordo'
import { useNotify } from '@/composables/useNotify'
import { wailsErrorCode } from '@/platform/wails/client'
import type { CTeRow } from '@/types/desktop'

// useCTeDesacordoActions holds the prestação em desacordo flow of the CT-e
// page: the dialog that plans it and the notifications of the answer. The
// requests and the in-flight marker go through useCTeDesacordo.
export function useCTeDesacordoActions() {
  const $q = useQuasar()
  const { notifyError, notifyWarning, notifyEventResult } = useNotify()
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
      notifyEventResult(result, 'prestação em desacordo')
    } catch (error) {
      if (wailsErrorCode(error) === 'canceled') {
        notifyWarning('Envio da prestação em desacordo cancelado.')
      } else {
        notifyError('Erro ao registrar a prestação em desacordo', error)
      }
    }
  }

  return { openDesacordo, isSending: desacordo.isSending }
}
