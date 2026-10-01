import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DocumentTableTop from './DocumentTableTop.vue'

const stubs = {
  QInput: {
    name: 'QInput',
    props: ['modelValue', 'placeholder'],
    emits: ['update:modelValue'],
    template: '<div class="search"><slot name="append" /></div>',
  },
  QIcon: { template: '<i />' },
}

describe('DocumentTableTop', () => {
  it('shows the title and the standard search box', () => {
    const wrapper = mount(DocumentTableTop, {
      props: { title: 'Notas fiscais eletrônicas', modelValue: 'acme' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('Notas fiscais eletrônicas')
    const input = wrapper.getComponent({ name: 'QInput' })
    expect(input.props('modelValue')).toBe('acme')
    expect(input.props('placeholder')).toBe('Filtrar por chave, número, nome ou CNPJ...')
    expect(input.classes()).toContain('document-search-input')
  })

  it('updates the filter text', async () => {
    const wrapper = mount(DocumentTableTop, {
      props: { title: 'CT-e', modelValue: '' },
      global: { stubs },
    })

    await wrapper.getComponent({ name: 'QInput' }).vm.$emit('update:modelValue', 'serra')

    expect(wrapper.emitted('update:modelValue')).toEqual([['serra']])
  })
})
