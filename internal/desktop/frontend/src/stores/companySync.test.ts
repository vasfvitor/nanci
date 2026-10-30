import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { useCompanySyncStore } from './companySync'
import { desktopClient } from '@/platform/wails/client'
import { deferred } from '@/test/fixtures'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { cancelSync: vi.fn() },
}))

describe('company sync store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('tracks active company syncs across store consumers', () => {
    const firstConsumer = useCompanySyncStore()
    firstConsumer.startSync('123', 'nfse')

    const secondConsumer = useCompanySyncStore()

    expect(secondConsumer.isSyncing('123', 'nfse')).toBe(true)
    expect(secondConsumer.activeSyncs).toEqual({ 'nfse:123': 1 })
  })

  it('says whether any sync runs', () => {
    const store = useCompanySyncStore()
    expect(store.anySyncing).toBe(false)

    store.startSync('123', 'nfe')
    expect(store.anySyncing).toBe(true)

    store.finishSync('123', 'nfe')
    expect(store.anySyncing).toBe(false)
  })

  it('keeps a company marked as syncing until all matching runs finish', () => {
    const store = useCompanySyncStore()

    store.startSync('123', 'nfse')
    store.startSync('123', 'nfse')
    store.finishSync('123', 'nfse')

    expect(store.isSyncing('123', 'nfse')).toBe(true)

    store.finishSync('123', 'nfse')

    expect(store.isSyncing('123', 'nfse')).toBe(false)
    expect(store.activeSyncs).toEqual({})
  })

  it('tracks NFS-e and NF-e syncs for the same company independently', () => {
    const store = useCompanySyncStore()

    store.startSync('123', 'nfe')

    expect(store.isSyncing('123', 'nfe')).toBe(true)
    expect(store.isSyncing('123', 'nfse')).toBe(false)

    store.startSync('123', 'nfse')
    store.finishSync('123', 'nfe')

    expect(store.isSyncing('123', 'nfe')).toBe(false)
    expect(store.isSyncing('123', 'nfse')).toBe(true)
  })

  it('keys CT-e syncs apart from NF-e syncs', () => {
    const store = useCompanySyncStore()

    store.startSync('123', 'cte')

    expect(store.isSyncing('123', 'cte')).toBe(true)
    expect(store.isSyncing('123', 'nfe')).toBe(false)
    expect(store.activeSyncs).toEqual({ 'cte:123': 1 })
  })

  it('ignores finishing a sync that was never started', () => {
    const store = useCompanySyncStore()

    store.startSync('123', 'nfse')
    store.finishSync('123', 'nfe')

    expect(store.isSyncing('123', 'nfse')).toBe(true)
    expect(store.activeSyncs).toEqual({ 'nfse:123': 1 })
  })

  it('marks the sync while runSync runs, also when it fails', async () => {
    const store = useCompanySyncStore()
    let seen = false

    await expect(
      store.runSync('123', 'cte', async () => {
        seen = store.isSyncing('123', 'cte')
        return 'ok'
      })
    ).resolves.toBe('ok')
    expect(seen).toBe(true)
    expect(store.isSyncing('123', 'cte')).toBe(false)

    await expect(
      store.runSync('123', 'cte', async () => {
        throw new Error('offline')
      })
    ).rejects.toThrow('offline')
    expect(store.isSyncing('123', 'cte')).toBe(false)
  })

  describe('cancelSync', () => {
    beforeEach(() => {
      vi.mocked(desktopClient.cancelSync).mockReset().mockResolvedValue(undefined)
    })

    it('asks the backend to stop and stays cancelando, from any consumer, until the call settles', async () => {
      const store = useCompanySyncStore()
      const pull = deferred<string>()
      const running = store.runCancellableSync('123', 'nfe', () => pull.promise)
      expect(store.canCancelSync('123', 'nfe')).toBe(true)
      expect(store.canCancelSync('123', 'cte')).toBe(false)

      await useCompanySyncStore().cancelSync('123', 'nfe')

      expect(desktopClient.cancelSync).toHaveBeenCalledWith('123', 'nfe')
      const other = useCompanySyncStore()
      expect(other.canCancelSync('123', 'nfe')).toBe(false)
      expect(other.isCancelling('123', 'nfe')).toBe(true)
      expect(other.isSyncing('123', 'nfe')).toBe(true)
      expect(other.anySyncing).toBe(true)

      await other.cancelSync('123', 'nfe')
      expect(desktopClient.cancelSync).toHaveBeenCalledTimes(1)

      pull.reject(new Error('cancelada'))
      await expect(running).rejects.toThrow('cancelada')
      expect(other.isCancelling('123', 'nfe')).toBe(false)
      expect(other.isSyncing('123', 'nfe')).toBe(false)
      expect(other.anySyncing).toBe(false)
    })

    it('leaves the sync cancellable when the backend call fails', async () => {
      const store = useCompanySyncStore()
      const pull = deferred<string>()
      const running = store.runCancellableSync('123', 'nfe', () => pull.promise)
      vi.mocked(desktopClient.cancelSync).mockRejectedValueOnce(new Error('offline'))

      await expect(store.cancelSync('123', 'nfe')).rejects.toThrow('offline')

      expect(store.isCancelling('123', 'nfe')).toBe(false)
      expect(store.canCancelSync('123', 'nfe')).toBe(true)
      pull.resolve('ok')
      await expect(running).resolves.toBe('ok')
    })

    it('offers no cancel for a plain runSync and ignores a cancel with nothing running', async () => {
      const store = useCompanySyncStore()
      let canCancel = true

      await store.runSync('123', 'nfe', async () => {
        canCancel = store.canCancelSync('123', 'nfe')
      })
      await store.cancelSync('123', 'nfe')

      expect(canCancel).toBe(false)
      expect(store.isCancelling('123', 'nfe')).toBe(false)
      expect(desktopClient.cancelSync).not.toHaveBeenCalled()
    })
  })
})
