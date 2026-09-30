import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AbbrBadge from './AbbrBadge.vue'

const quasar = vi.hoisted(() => ({ dark: { isActive: false } }))

vi.mock('quasar', () => ({ useQuasar: () => quasar }))

function mountBadge(props: InstanceType<typeof AbbrBadge>['$props']) {
  return mount(AbbrBadge, {
    props,
    global: {
      stubs: {
        QBadge: {
          props: ['label', 'color', 'textColor'],
          template:
            '<span class="badge" :data-color="color" :data-text-color="textColor">{{ label }}<slot /></span>',
        },
        QTooltip: { template: '<span class="tooltip"><slot /></span>' },
      },
    },
  })
}

describe('AbbrBadge', () => {
  beforeEach(() => {
    quasar.dark.isActive = false
  })

  it('shows the abbreviation and explains it as "Kind: Label"', () => {
    const wrapper = mountBadge({ abbr: 'A', label: 'Autorizada', color: 'positive', kind: 'Situação' })

    const badge = wrapper.find('.badge')
    expect(badge.text()).toContain('A')
    expect(wrapper.find('.tooltip').text()).toBe('Situação: Autorizada')
    expect(badge.attributes('aria-label')).toBe('Situação: Autorizada')
    expect(badge.classes()).toContain('text-mono')
    expect(badge.classes()).not.toContain('abbr-badge--secondary')
  })

  it('explains the badge by its label alone without a kind', () => {
    const wrapper = mountBadge({ abbr: 'N', label: 'Normal', color: 'positive' })
    expect(wrapper.find('.tooltip').text()).toBe('Normal')
  })

  it('marks secondary badges', () => {
    const wrapper = mountBadge({
      abbr: 'RE',
      label: 'Remetente',
      color: 'info',
      kind: 'Papel',
      secondary: true,
    })
    expect(wrapper.find('.badge').classes()).toContain('abbr-badge--secondary')
  })

  it('binds readable colors in the light theme', () => {
    const badge = mountBadge({ abbr: 'CI', label: 'Ciência', color: 'info' }).find('.badge')
    expect(badge.attributes('data-color')).toBe('light-blue-9')
    expect(badge.attributes('data-text-color')).toBe('white')
  })

  it('binds readable colors in the dark theme', () => {
    quasar.dark.isActive = true
    const badge = mountBadge({ abbr: 'CI', label: 'Ciência', color: 'info' }).find('.badge')
    expect(badge.attributes('data-color')).toBe('info')
    expect(badge.attributes('data-text-color')).toBe('dark')
  })
})
