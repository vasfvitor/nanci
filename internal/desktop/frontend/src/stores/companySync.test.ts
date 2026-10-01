import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect } from 'vitest'
import { useCompanySyncStore } from './companySync'

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
})
