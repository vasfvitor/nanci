import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useFormDraft } from './useFormDraft'
import { useFormDraftsStore } from '@/stores/formDrafts'

describe('useFormDraft', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('restores a dirty draft and flags it', () => {
    const first = useFormDraft('company:add', () => ({ Name: '' }))
    first.open()
    first.form.value.Name = 'Empresa Um'
    expect(first.dirty.value).toBe(true)
    first.keep()

    // A second instance stands for the dialog mounted again.
    const second = useFormDraft('company:add', () => ({ Name: '' }))
    second.open()
    expect(second.form.value.Name).toBe('Empresa Um')
    expect(second.restored.value).toBe(true)
    expect(second.dirty.value).toBe(true)
  })

  it('drops an untouched form on close', () => {
    const store = useFormDraftsStore()
    const draft = useFormDraft('company:add', () => ({ Name: '' }))
    draft.open()
    draft.form.value.Name = 'Empresa Um'
    draft.keep()
    expect(store.get('company:add')).toBeDefined()

    draft.open()
    draft.form.value.Name = ''
    draft.keep()
    expect(store.get('company:add')).toBeUndefined()

    draft.open()
    expect(draft.restored.value).toBe(false)
  })

  it('drops a draft whose seed changed', () => {
    const store = useFormDraftsStore()
    const name = ref('Empresa Um')
    const draft = useFormDraft('company:edit:1', () => ({ Name: name.value }), {
      dropWhenSeedChanges: true,
    })
    draft.open()
    draft.form.value.Name = 'Empresa Editada'
    draft.keep()

    // The company was renamed elsewhere after the draft was kept.
    name.value = 'Empresa Renomeada'
    draft.open()
    expect(draft.form.value.Name).toBe('Empresa Renomeada')
    expect(draft.restored.value).toBe(false)
    expect(store.get('company:edit:1')).toBeUndefined()
  })

  it('clear resets to the seed', () => {
    const store = useFormDraftsStore()
    const draft = useFormDraft('credential:add', () => ({ Label: '' }))
    draft.open()
    draft.form.value.Label = 'A'
    draft.keep()
    draft.open()
    expect(draft.restored.value).toBe(true)

    draft.clear()
    expect(draft.form.value.Label).toBe('')
    expect(draft.restored.value).toBe(false)
    expect(draft.dirty.value).toBe(false)
    expect(store.get('credential:add')).toBeUndefined()
  })

  it('done leaves no draft', () => {
    const store = useFormDraftsStore()
    const draft = useFormDraft('credential:add', () => ({ Label: '' }))
    draft.open()
    draft.form.value.Label = 'A'
    draft.done()
    expect(draft.form.value.Label).toBe('')
    draft.keep()
    expect(store.get('credential:add')).toBeUndefined()
  })
})
