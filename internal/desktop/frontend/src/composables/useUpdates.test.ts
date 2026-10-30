import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { updateBlockedBySync, useUpdates } from './useUpdates'
import { desktopClient, WailsClientError } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import type { UpdateInfo } from '@/types/desktop'

type NotifyAction = { label: string; handler?: () => void }
type NotifyOptions = { type: string; message: string; timeout?: number; actions?: NotifyAction[] }

const notify = vi.fn((_options: NotifyOptions) => undefined)

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
  copyToClipboard: vi.fn(),
  Notify: { create: vi.fn() },
}))

vi.mock('@/platform/wails/client', async () => {
  const base = await vi.importActual<typeof import('@/platform/wails/client/base')>(
    '@/platform/wails/client/base'
  )
  return {
    desktopClient: { checkForUpdate: vi.fn(), installUpdate: vi.fn(), showInFolder: vi.fn() },
    errorMessage: base.errorMessage,
    wailsErrorCode: base.wailsErrorCode,
    WailsClientError: base.WailsClientError,
  }
})

vi.mock('@/platform/wails/events', () => ({
  onWailsEvent: () => vi.fn(),
}))

const found: UpdateInfo = {
  enabled: true,
  currentVersion: '0.6.0',
  available: true,
  version: '0.7.0',
  notes: '',
}

function lastNotice(): NotifyOptions {
  const options = notify.mock.calls.at(-1)?.[0]
  if (!options) throw new Error('no notice')
  return options
}

function noticeAction(label: string) {
  const action = lastNotice().actions?.find((a) => a.label === label)
  if (!action?.handler) throw new Error(`no action ${label}`)
  return action.handler
}

describe('useUpdates', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('offers a release found at startup and installs it from the notice', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(found)
    vi.mocked(desktopClient.installUpdate).mockResolvedValue(undefined)
    const updates = useUpdates()

    await updates.checkAtStartup()

    expect(lastNotice()).toMatchObject({
      type: 'info',
      message: 'Nova versão 0.7.0 disponível.',
      timeout: 0,
    })
    noticeAction('Atualizar e reiniciar')()
    await vi.waitFor(() => expect(updates.status.value).toBe('installing'))
    expect(desktopClient.installUpdate).toHaveBeenCalledTimes(1)
  })

  it('says nothing at startup when there is no release or the check fails', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockRejectedValue(new WailsClientError('sem rede'))

    await useUpdates().checkAtStartup()

    expect(notify).not.toHaveBeenCalled()
  })

  it('refuses the notice action while a sync runs, without a backend call', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(found)
    const updates = useUpdates()
    await updates.checkAtStartup()
    useCompanySyncStore().startSync('12345678000195', 'nfe')

    noticeAction('Atualizar e reiniciar')()

    expect(updates.blockedBySync.value).toBe(true)
    expect(lastNotice()).toMatchObject({ type: 'warning', message: updateBlockedBySync })
    expect(desktopClient.installUpdate).not.toHaveBeenCalled()
  })

  it('warns when the backend refuses because a sync started', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(found)
    vi.mocked(desktopClient.installUpdate).mockRejectedValue(
      new WailsClientError('aguarde a sincronização terminar para atualizar', 'sync_running')
    )
    const updates = useUpdates()
    await updates.checkNow()

    await updates.installAndRestart()

    expect(lastNotice()).toMatchObject({ type: 'warning', message: updateBlockedBySync })
    expect(updates.status.value).toBe('available')
  })

  it('reports a failed install', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(found)
    vi.mocked(desktopClient.installUpdate).mockRejectedValue(new WailsClientError('falhou'))
    const updates = useUpdates()
    await updates.checkNow()

    await updates.installAndRestart()

    expect(lastNotice()).toMatchObject({ type: 'negative', message: 'Erro ao atualizar: falhou' })
    expect(updates.statusLabel.value).toBe('erro: falhou')
  })

  it('confirms an up-to-date check from the button', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue({ ...found, available: false })
    const updates = useUpdates()

    await updates.checkNow()

    expect(lastNotice()).toMatchObject({
      type: 'positive',
      message: 'O Nanci está na versão mais recente.',
    })
    expect(updates.statusLabel.value).toBe('versão mais recente')
  })

  it('reports a failed check from the button', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockRejectedValue(new WailsClientError('sem rede'))

    await useUpdates().checkNow()

    expect(lastNotice()).toMatchObject({
      type: 'negative',
      message: 'Erro ao procurar atualizações: sem rede',
    })
  })

  it('labels a build without updates', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue({
      ...found,
      enabled: false,
      available: false,
      version: '',
    })
    const updates = useUpdates()

    await updates.checkNow()

    expect(notify).not.toHaveBeenCalled()
    expect(updates.statusLabel.value).toBe('indisponíveis nesta versão')
  })
})
