import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, expect, vi } from 'vitest'
import { CANCEL_POLL_MS, CANCEL_WAIT_MS, useCompanySyncStore } from './companySync'

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
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    function runAbortable(store: ReturnType<typeof useCompanySyncStore>) {
      const signals: AbortSignal[] = []
      const running = store.runCancellableSync(
        '123',
        'nfe',
        (signal) =>
          new Promise((_resolve, reject) => {
            signals.push(signal)
            signal.addEventListener('abort', () => reject(new Error('cancelada')))
          })
      )
      return { running, signals }
    }

    it('keeps the sync cancelando, from any consumer, until the backend is done', async () => {
      const store = useCompanySyncStore()
      const { running, signals } = runAbortable(store)
      expect(store.canCancelSync('123', 'nfe')).toBe(true)
      expect(store.canCancelSync('123', 'cte')).toBe(false)
      let backendRunning = true
      const stillRunning = vi.fn(async () => backendRunning)

      const cancelled = useCompanySyncStore().cancelSync('123', 'nfe', stillRunning)

      expect(signals[0]?.aborted).toBe(true)
      await expect(running).rejects.toThrow('cancelada')
      const other = useCompanySyncStore()
      expect(other.canCancelSync('123', 'nfe')).toBe(false)
      expect(other.isCancelling('123', 'nfe')).toBe(true)
      expect(other.isSyncing('123', 'nfe')).toBe(true)
      expect(other.anySyncing).toBe(true)

      await vi.advanceTimersByTimeAsync(CANCEL_POLL_MS * 3)
      expect(stillRunning).toHaveBeenCalledTimes(3)
      expect(other.isCancelling('123', 'nfe')).toBe(true)

      backendRunning = false
      await vi.advanceTimersByTimeAsync(CANCEL_POLL_MS)
      await cancelled
      expect(other.isCancelling('123', 'nfe')).toBe(false)
      expect(other.isSyncing('123', 'nfe')).toBe(false)
      expect(other.anySyncing).toBe(false)
    })

    it('gives up waiting after CANCEL_WAIT_MS, also when the check fails', async () => {
      const store = useCompanySyncStore()
      const { running } = runAbortable(store)
      const stillRunning = vi.fn(async () => {
        throw new Error('offline')
      })

      const cancelled = store.cancelSync('123', 'nfe', stillRunning)
      await expect(running).rejects.toThrow('cancelada')

      await vi.advanceTimersByTimeAsync(CANCEL_WAIT_MS - CANCEL_POLL_MS)
      expect(store.isCancelling('123', 'nfe')).toBe(true)
      await vi.advanceTimersByTimeAsync(CANCEL_POLL_MS)
      await cancelled
      expect(store.isCancelling('123', 'nfe')).toBe(false)
      expect(store.isSyncing('123', 'nfe')).toBe(false)
    })

    it('offers no cancel for a plain runSync and ignores a cancel with nothing running', async () => {
      const store = useCompanySyncStore()
      let canCancel = true
      const stillRunning = vi.fn(async () => true)

      await store.runSync('123', 'nfe', async () => {
        canCancel = store.canCancelSync('123', 'nfe')
      })
      await store.cancelSync('123', 'nfe', stillRunning)

      expect(canCancel).toBe(false)
      expect(store.isCancelling('123', 'nfe')).toBe(false)
      expect(stillRunning).not.toHaveBeenCalled()
    })
  })
})
