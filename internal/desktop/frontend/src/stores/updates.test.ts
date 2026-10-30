import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useUpdatesStore } from './updates'
import { desktopClient, WailsClientError } from '@/platform/wails/client'
import { deferred } from '@/test/fixtures'
import type { UpdateInfo } from '@/types/desktop'

vi.mock('@/platform/wails/client', async () => {
  const base = await vi.importActual<typeof import('@/platform/wails/client/base')>(
    '@/platform/wails/client/base'
  )
  return {
    desktopClient: { checkForUpdate: vi.fn(), installUpdate: vi.fn() },
    errorMessage: base.errorMessage,
    wailsErrorCode: base.wailsErrorCode,
    WailsClientError: base.WailsClientError,
  }
})

type Handler = (payload: unknown) => void
const handlers = new Map<string, Handler>()
const unsubscribe = vi.fn()

vi.mock('@/platform/wails/events', () => ({
  onWailsEvent: (eventName: string, callback: Handler) => {
    handlers.set(eventName, callback)
    return unsubscribe
  },
}))

function updateInfo(fields: Partial<UpdateInfo> = {}): UpdateInfo {
  return {
    enabled: true,
    currentVersion: '0.6.0',
    available: false,
    version: '',
    notes: '',
    ...fields,
  }
}

const found = updateInfo({ available: true, version: '0.7.0', notes: 'Novidades' })

async function storeWithUpdate() {
  vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(found)
  const store = useUpdatesStore()
  await store.check()
  return store
}

describe('updates store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    handlers.clear()
  })

  it('marks a found release as available', async () => {
    const store = await storeWithUpdate()

    expect(store.status).toBe('available')
    expect(store.info).toEqual(found)
    expect(store.canInstall).toBe(true)
  })

  it('goes back to idle when the app is up to date', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(updateInfo())
    const store = useUpdatesStore()

    await store.check()

    expect(store.status).toBe('idle')
    expect(store.canInstall).toBe(false)
  })

  it('is unavailable in a build that does not update itself', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockResolvedValue(
      updateInfo({ enabled: false, currentVersion: '0.0.0-dev' })
    )
    const store = useUpdatesStore()

    await store.check()

    expect(store.status).toBe('unavailable')
  })

  it('keeps the message of a failed check and rethrows it', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockRejectedValue(new WailsClientError('sem rede'))
    const store = useUpdatesStore()

    await expect(store.check()).rejects.toThrow('sem rede')
    expect(store.status).toBe('error')
    expect(store.error).toBe('sem rede')
  })

  it('checks once at startup, even across remounts, and swallows the failure', async () => {
    vi.mocked(desktopClient.checkForUpdate).mockRejectedValue(new WailsClientError('sem rede'))
    const store = useUpdatesStore()

    const first = store.checkOnce()
    const second = useUpdatesStore().checkOnce()

    await expect(first).resolves.toBeNull()
    await expect(second).resolves.toBeNull()
    await expect(store.checkOnce()).resolves.toBeNull()
    expect(desktopClient.checkForUpdate).toHaveBeenCalledTimes(1)
    expect(store.error).toBe('sem rede')
  })

  it('does not install without a release', async () => {
    const store = useUpdatesStore()

    await expect(store.install()).resolves.toBe(false)
    expect(desktopClient.installUpdate).not.toHaveBeenCalled()
  })

  it('keeps a download visible to a second instance while it is pending', async () => {
    const store = await storeWithUpdate()
    const call = deferred<undefined>()
    vi.mocked(desktopClient.installUpdate).mockReturnValue(call.promise)

    const running = store.install()
    const again = useUpdatesStore()

    expect(again.status).toBe('downloading')
    expect(again.busy).toBe(true)
    await expect(again.install()).resolves.toBe(false)
    await expect(again.check()).resolves.toBeNull()
    expect(desktopClient.installUpdate).toHaveBeenCalledTimes(1)
    expect(desktopClient.checkForUpdate).toHaveBeenCalledTimes(1)

    call.resolve(undefined)
    await expect(running).resolves.toBe(true)
    expect(again.status).toBe('installing')
    expect(unsubscribe).toHaveBeenCalledTimes(1)
  })

  it('follows the download progress until the whole file is in', async () => {
    const store = await storeWithUpdate()
    const call = deferred<undefined>()
    vi.mocked(desktopClient.installUpdate).mockReturnValue(call.promise)

    const running = store.install()
    handlers.get('wails:updater:download-progress')?.({ written: 25, total: 100, rate: 10 })
    expect(store.percent).toBe(25)
    expect(store.status).toBe('downloading')

    handlers.get('wails:updater:download-progress')?.({ written: 100, total: 100, rate: 10 })
    expect(store.percent).toBe(100)
    expect(store.status).toBe('installing')

    call.resolve(undefined)
    await running
  })

  it('has no percent while the size is unknown', async () => {
    const store = await storeWithUpdate()
    const call = deferred<undefined>()
    vi.mocked(desktopClient.installUpdate).mockReturnValue(call.promise)

    const running = store.install()
    handlers.get('wails:updater:download-progress')?.({ written: 25, total: 0, rate: 10 })

    expect(store.percent).toBeNull()
    call.resolve(undefined)
    await running
  })

  it('puts the release back when a sync runs', async () => {
    const store = await storeWithUpdate()
    vi.mocked(desktopClient.installUpdate).mockRejectedValue(
      new WailsClientError('aguarde a sincronização terminar para atualizar', 'sync_running')
    )

    await expect(store.install()).rejects.toThrow('aguarde')

    expect(store.status).toBe('available')
    expect(store.error).toBe('')
    expect(unsubscribe).toHaveBeenCalledTimes(1)
  })

  it('shows the refusal while another operation runs as an error', async () => {
    const store = await storeWithUpdate()
    vi.mocked(desktopClient.installUpdate).mockRejectedValue(
      new WailsClientError('aguarde a operação em andamento terminar para atualizar')
    )

    await expect(store.install()).rejects.toThrow('aguarde')

    expect(store.status).toBe('error')
    expect(store.error).toBe('aguarde a operação em andamento terminar para atualizar')
    expect(store.canInstall).toBe(true)
  })

  it('keeps a failed install retryable', async () => {
    const store = await storeWithUpdate()
    vi.mocked(desktopClient.installUpdate).mockRejectedValue(
      new WailsClientError('baixar atualização: updater: ed25519ph signature did not verify')
    )

    await expect(store.install()).rejects.toThrow('signature')

    expect(store.status).toBe('error')
    expect(store.error).toContain('signature did not verify')
    expect(store.canInstall).toBe(true)
  })
})
