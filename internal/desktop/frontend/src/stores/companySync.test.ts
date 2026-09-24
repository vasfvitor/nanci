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

  it('ignores finishing a sync that was never started', () => {
    const store = useCompanySyncStore()

    store.startSync('123', 'nfse')
    store.finishSync('123', 'nfe')

    expect(store.isSyncing('123', 'nfse')).toBe(true)
    expect(store.activeSyncs).toEqual({ 'nfse:123': 1 })
  })
})
