import { computed } from 'vue'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import { wailsErrorCode } from '@/platform/wails/client'
import { useNotify } from '@/composables/useNotify'
import { useCompanySyncStore } from '@/stores/companySync'
import { useUpdatesStore } from '@/stores/updates'
import type { UpdateInfo } from '@/types/desktop'

export const updateBlockedBySync =
  'Aguarde a sincronização terminar para atualizar: a atualização reinicia o Nanci.'

// useUpdates offers the update at startup and from the Sobre section. The
// restart never runs during a sync: the action is refused here, and the
// backend refuses it again.
export function useUpdates() {
  const $q = useQuasar()
  const { notifyError, notifySuccess, notifyWarning } = useNotify()
  const store = useUpdatesStore()
  const companySync = useCompanySyncStore()
  const { status, info, error, busy, canInstall, percent } = storeToRefs(store)
  const blockedBySync = computed(() => companySync.anySyncing)
  // installing covers the download and the restart.
  const installing = computed(() => status.value === 'downloading' || status.value === 'installing')
  // statusLabel is the state shown in the Sobre section.
  const statusLabel = computed(() => {
    switch (status.value) {
      case 'unavailable':
        return 'indisponíveis nesta versão'
      case 'checking':
        return 'procurando...'
      case 'available':
        return `versão ${info.value?.version ?? ''} disponível`
      case 'downloading':
        return percent.value === null ? 'baixando...' : `baixando (${percent.value}%)`
      case 'installing':
        return 'instalando; o Nanci vai reiniciar'
      case 'error':
        return `erro: ${error.value}`
      default:
        return info.value?.enabled ? 'versão mais recente' : '...'
    }
  })

  async function installAndRestart() {
    if (blockedBySync.value) {
      notifyWarning(updateBlockedBySync)
      return
    }
    store.replaceNotice(null)
    try {
      await store.install()
    } catch (e) {
      if (wailsErrorCode(e) === 'sync_running') notifyWarning(updateBlockedBySync)
      else notifyError('Erro ao atualizar', e)
    }
  }

  // notifyAvailable stays until the user acts on it, an install starts or a
  // later check replaces it.
  function notifyAvailable(found: UpdateInfo) {
    const dismiss = $q.notify({
      type: 'info',
      message: `Nova versão ${found.version} disponível.`,
      timeout: 0,
      actions: [
        { label: 'Atualizar e reiniciar', color: 'white', handler: () => void installAndRestart() },
        { label: 'Depois', color: 'white' },
      ],
    })
    store.replaceNotice(dismiss)
  }

  // checkAtStartup runs the session's single startup check and offers what
  // it finds. A failure is silent here; the Sobre section shows it.
  async function checkAtStartup() {
    const found = await store.checkOnce()
    if (found?.available) notifyAvailable(found)
  }

  // checkNow is the "Procurar atualizações" button.
  async function checkNow() {
    try {
      const found = await store.check()
      if (!found) return
      if (found.available) {
        notifyAvailable(found)
        return
      }
      store.replaceNotice(null)
      if (found.enabled) notifySuccess('O Nanci está na versão mais recente.')
    } catch (e) {
      notifyError('Erro ao procurar atualizações', e)
    }
  }

  return {
    status,
    info,
    error,
    busy,
    canInstall,
    percent,
    installing,
    statusLabel,
    blockedBySync,
    installAndRestart,
    checkAtStartup,
    checkNow,
  }
}
