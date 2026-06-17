import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect } from 'vitest'
import { useCompanySyncStore } from './companySync'

describe('company sync store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('tracks active company syncs across store consumers', () => {
    const firstConsumer = useCompanySyncStore()
    firstConsumer.startSync('123')

    const secondConsumer = useCompanySyncStore()

    expect(secondConsumer.isSyncing('123')).toBe(true)
    expect(secondConsumer.syncing).toBe('123')
    expect(secondConsumer.syncingCNPJs).toEqual(['123'])
  })

  it('keeps a company marked as syncing until all matching runs finish', () => {
    const store = useCompanySyncStore()

    store.startSync('123')
    store.startSync('123')
    store.finishSync('123')

    expect(store.isSyncing('123')).toBe(true)

    store.finishSync('123')

    expect(store.isSyncing('123')).toBe(false)
    expect(store.syncing).toBeNull()
  })
})
