import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PartyCell from './PartyCell.vue'

describe('PartyCell', () => {
  it('shows the name over the formatted CNPJ, both in the title', () => {
    const wrapper = mount(PartyCell, { props: { name: 'Fornecedor Ltda', cnpj: '12345678000199' } })

    expect(wrapper.find('.party-name').text()).toBe('Fornecedor Ltda')
    const cnpj = wrapper.find('.party-cnpj')
    expect(cnpj.text()).toBe('12.345.678/0001-99')
    expect(cnpj.classes()).toEqual(expect.arrayContaining(['text-app-muted', 'text-mono']))
    expect(wrapper.attributes('title')).toBe('Fornecedor Ltda\n12.345.678/0001-99')
  })

  it('falls back to a dash for missing values', () => {
    const wrapper = mount(PartyCell, { props: { name: '', cnpj: '' } })

    expect(wrapper.find('.party-name').text()).toBe('—')
    expect(wrapper.find('.party-cnpj').text()).toBe('—')
    expect(wrapper.attributes('title')).toBe('')
  })
})
