import { useQuasar } from 'quasar'
import { desktopClient } from '@/platform/wails/client'
import { onWailsEvent, type Unsubscribe } from '@/platform/wails/events'
import { useNotify } from '@/composables/useNotify'

// useCloseGuard asks before the window closes while a sync runs. The backend
// keeps the window open and sends app:before-close; leaving calls
// confirmQuit. start returns the unsubscribe of the event.
export function useCloseGuard() {
  const $q = useQuasar()
  const { notifyError } = useNotify()
  let prompting = false

  function ask() {
    if (prompting) return
    prompting = true
    $q.dialog({
      title: 'Sincronização em andamento',
      message:
        'Há uma sincronização em andamento. Se sair agora, ela será interrompida; os documentos já salvos são mantidos. Deseja sair mesmo assim?',
      ok: { label: 'Sair', color: 'negative', flat: true },
      cancel: { label: 'Continuar no Nanci', flat: true },
      persistent: true,
    })
      .onOk(() => {
        desktopClient.confirmQuit().catch((e: unknown) => notifyError('Erro ao sair', e))
      })
      .onDismiss(() => {
        prompting = false
      })
  }

  function start(): Unsubscribe {
    return onWailsEvent('app:before-close', ask)
  }

  return { start }
}
