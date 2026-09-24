import { copyToClipboard, useQuasar } from 'quasar'
import { errorMessage, wailsErrorCode } from '@/platform/wails/client'

// useNotify holds the notifications the document pages share.
export function useNotify() {
  const $q = useQuasar()

  function notifyError(message: string, error: unknown) {
    $q.notify({ type: 'negative', message: `${message}: ${errorMessage(error)}` })
  }

  // notifySyncError reports a failed sync. A canceled, already running or
  // SEFAZ-blocked sync is a warning; anything else is an error under message.
  function notifySyncError(message: string, error: unknown) {
    switch (wailsErrorCode(error)) {
      case 'canceled':
        $q.notify({ type: 'warning', message: 'Sincronização cancelada.' })
        return
      case 'sync_running':
        $q.notify({ type: 'warning', message: 'Sincronização já em andamento para esta empresa.' })
        return
      case 'sefaz_blocked':
        $q.notify({
          type: 'warning',
          message: 'Consultas bloqueadas no momento. Aguarde o horário indicado.',
        })
        return
      default:
        notifyError(message, error)
    }
  }

  // copyChave copies an access key. NFS-e keys lose their NFS prefix, so the
  // copied value is the key the portal accepts.
  async function copyChave(chave: string | null | undefined) {
    if (!chave) return
    try {
      await copyToClipboard(chave.replace(/^NFS/i, ''))
      $q.notify({ type: 'positive', message: 'Chave copiada!', timeout: 1000 })
    } catch (error) {
      notifyError('Erro ao copiar chave', error)
    }
  }

  return { notifyError, notifySyncError, copyChave }
}
