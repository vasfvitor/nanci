import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useFormDraftsStore } from './formDrafts'

describe('formDrafts store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('stores, reads and drops drafts by key', () => {
    const store = useFormDraftsStore()
    expect(store.get('company:add')).toBeUndefined()

    store.set('company:add', { seed: '{}', value: '{"Name":"A"}' })
    store.set('credential:add', { seed: '{}', value: '{"Label":"B"}' })
    expect(store.get('company:add')).toEqual({ seed: '{}', value: '{"Name":"A"}' })

    store.drop('company:add')
    expect(store.get('company:add')).toBeUndefined()
    expect(store.get('credential:add')).toEqual({ seed: '{}', value: '{"Label":"B"}' })
  })
})
