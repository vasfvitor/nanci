import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import NumeroCell from './NumeroCell.vue'

vi.mock('quasar', () => ({ useQuasar: () => ({ dark: { isActive: false } }) }))

const stubs = {
  QBadge: { props: ['label', 'color'], template: '<span class="badge" :data-color="color">{{ label }}</span>' },
}

describe('NumeroCell', () => {
  it('shows the number, the série and the "Novo" badge of a document not viewed', () => {
    const wrapper = mount(NumeroCell, {
      props: { numero: '000.018.736', serie: '1', viewedAt: null },
      global: { stubs },
    })

    expect(wrapper.find('.numero-cell').text()).toBe('000.018.736')
    expect(wrapper.text()).toContain('série 1')
    expect(wrapper.find('.badge').text()).toBe('Novo')
    expect(wrapper.find('.badge').attributes('data-color')).toBe('warning')
    expect(wrapper.attributes('title')).toBe('000.018.736 / 1')
  })

  it('shows only the number of a viewed document without série', () => {
    const wrapper = mount(NumeroCell, {
      props: { numero: '123', viewedAt: '2026-06-01T10:00:00Z' },
      global: { stubs },
    })

    expect(wrapper.find('.numero-cell').text()).toBe('123')
    expect(wrapper.find('.badge').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('série')
    expect(wrapper.attributes('title')).toBe('123')
  })

  it('falls back to a dash without a number', () => {
    const wrapper = mount(NumeroCell, { props: { numero: '', viewedAt: new Date() }, global: { stubs } })

    expect(wrapper.find('.numero-cell').text()).toBe('—')
  })
})
