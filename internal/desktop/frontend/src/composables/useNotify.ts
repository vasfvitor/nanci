import { copyToClipboard, Notify, useQuasar, type QNotifyCreateOptions } from 'quasar'
import { errorMessage, wailsErrorCode } from '@/platform/wails/client'
import type { ExportResult } from '@/types/desktop'

type NotifyOptions = Omit<QNotifyCreateOptions, 'type' | 'message'>

// errorNotice is the notification of a failed action: "message: cause".
function errorNotice(message: string, error: unknown): QNotifyCreateOptions {
  return { type: 'negative', message: `${message}: ${errorMessage(error)}` }
}

// copyChave copies an access key as stored. The NFS-e chave is stored as its
// 50 digits, the key the portal accepts. It needs no component, so a table
// cell can call it without holding the notify API.
export async function copyChave(chave: string | null | undefined) {
  if (!chave) return
  try {
    await copyToClipboard(chave)
    Notify.create({ type: 'positive', message: 'Chave copiada!', timeout: 1000 })
  } catch (error) {
    Notify.create(errorNotice('Erro ao copiar chave', error))
  }
}

// useNotify holds the notifications the document pages share.
export function useNotify() {
  const $q = useQuasar()

  function notifyError(message: string, error: unknown) {
    $q.notify(errorNotice(message, error))
  }

  function notifySuccess(message: string, options: NotifyOptions = {}) {
    $q.notify({ ...options, type: 'positive', message })
  }

  function notifyInfo(message: string, options: NotifyOptions = {}) {
    $q.notify({ ...options, type: 'info', message })
  }

  function notifyWarning(message: string, options: NotifyOptions = {}) {
    $q.notify({ ...options, type: 'warning', message })
  }

  // notifyExported reports an export: "N XMLs exportados para <caminho>.",
  // or that nothing was exported. noun is the singular name of the exported
  // files ("XML", "DANFSe", "documento"). A missing result means the export
  // did not run, and says nothing.
  function notifyExported(
    result: Pick<ExportResult, 'ExportedCount' | 'OutPath'> | null | undefined,
    noun = 'XML'
  ) {
    if (!result) return
    const count = result.ExportedCount
    if (count === 0) {
      notifyInfo('Nenhum documento para exportar.')
      return
    }
    const what = count === 1 ? `1 ${noun} exportado` : `${count} ${noun}s exportados`
    notifySuccess(`${what} para ${result.OutPath}.`)
  }

  // notifySyncError reports a failed sync. A canceled, already running or
  // SEFAZ-blocked sync is a warning; anything else is an error under message.
  function notifySyncError(message: string, error: unknown) {
    switch (wailsErrorCode(error)) {
      case 'canceled':
        notifyWarning('Sincronização cancelada.')
        return
      case 'sync_running':
        notifyWarning('Sincronização já em andamento para esta empresa.')
        return
      case 'sefaz_blocked':
        notifyWarning('Consultas bloqueadas no momento. Aguarde o horário indicado.')
        return
      default:
        notifyError(message, error)
    }
  }

  return {
    notifyError,
    notifySuccess,
    notifyInfo,
    notifyWarning,
    notifyExported,
    notifySyncError,
  }
}
