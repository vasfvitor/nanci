import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect } from 'vitest'
import { useQueryStore } from './query'

describe('query store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts with an empty NFS-e query and no chave suggestions', () => {
    const store = useQueryStore()

    expect(store.form).toEqual({ chave: '' })
    expect(store.result).toBe('')
    expect(store.type).toBe('nfse')
    expect(store.loading).toBe(false)
    expect(store.chaveOptions).toEqual([])
    expect(store.optionsCNPJ).toBe('')
  })

  it('keeps the form, the result and the suggestions for the next visit', () => {
    const first = useQueryStore()
    first.form.chave = '12345'
    first.type = 'events'
    first.result = '{"ok":true}'
    first.chaveOptions = [{ label: '12345', value: '12345' }]
    first.optionsCNPJ = '11111111000111'

    const again = useQueryStore()
    expect(again.form.chave).toBe('12345')
    expect(again.type).toBe('events')
    expect(again.result).toBe('{"ok":true}')
    expect(again.chaveOptions).toEqual([{ label: '12345', value: '12345' }])
    expect(again.optionsCNPJ).toBe('11111111000111')
  })

  it('clearResult clears only the result', () => {
    const store = useQueryStore()
    store.form.chave = '12345'
    store.type = 'events'
    store.result = '{"ok":true}'
    store.chaveOptions = [{ label: '12345', value: '12345' }]
    store.optionsCNPJ = '11111111000111'

    store.clearResult()

    expect(store.result).toBe('')
    expect(store.form.chave).toBe('12345')
    expect(store.type).toBe('events')
    expect(store.chaveOptions).toHaveLength(1)
    expect(store.optionsCNPJ).toBe('11111111000111')
  })

  it('shares one suggestion gate, so only the latest fetch counts', () => {
    const earlier = useQueryStore().optionsGate.begin()
    const latest = useQueryStore().optionsGate.begin()

    expect(earlier()).toBe(false)
    expect(latest()).toBe(true)
  })
})
