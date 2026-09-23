import { copyToClipboard, useQuasar } from 'quasar'
import { errorMessage } from '@/platform/wails/client'

// useNotify holds the notifications the document pages share.
export function useNotify() {
  const $q = useQuasar()

  function notifyError(message: string, error: unknown) {
    $q.notify({ type: 'negative', message: `${message}: ${errorMessage(error)}` })
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

  return { notifyError, copyChave }
}
