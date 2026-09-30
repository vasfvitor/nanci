import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StateBadges from './StateBadges.vue'
import type { StateBadge } from '@/utils/sefazDisplay'

vi.mock('quasar', () => ({ useQuasar: () => ({ dark: { isActive: false } }) }))

const stubs = {
  QBadge: { props: ['label'], template: '<span class="badge">{{ label }}</span>' },
  QTooltip: { template: '<span />' },
}

function badge(abbr: string, kind: string, secondary = false): StateBadge {
  return {
    key: `${kind}:${abbr}`,
    abbr,
    label: abbr,
    color: 'grey',
    kind,
    ...(secondary ? { secondary: true } : {}),
  }
}

describe('StateBadges', () => {
  it('shows the badges in order on one line', () => {
    const wrapper = mount(StateBadges, {
      props: { badges: [badge('CT', 'Documento'), badge('A', 'Situação'), badge('TO', 'Papel')] },
      global: { stubs },
    })

    expect(wrapper.findAll('.badge').map((el) => el.text())).toEqual(['CT', 'A', 'TO'])
    expect(wrapper.find('.state-badges-secondary').exists()).toBe(false)
  })

  it('puts secondary badges and the slot on a second line', () => {
    const wrapper = mount(StateBadges, {
      props: {
        badges: [badge('CT', 'Documento'), badge('TO', 'Papel'), badge('RE', 'Papel', true)],
      },
      slots: { default: '<span class="chip">5 d restantes</span>' },
      global: { stubs },
    })

    const secondLine = wrapper.find('.state-badges-secondary')
    expect(secondLine.findAll('.badge').map((el) => el.text())).toEqual(['RE'])
    expect(secondLine.find('.chip').text()).toBe('5 d restantes')
    expect(wrapper.findAll('.badge').map((el) => el.text())).toEqual(['CT', 'TO', 'RE'])
  })

  it('shows the second line for the slot alone', () => {
    const wrapper = mount(StateBadges, {
      props: { badges: [badge('A', 'Situação')] },
      slots: { default: '<span class="spinner" />' },
      global: { stubs },
    })

    expect(wrapper.find('.state-badges-secondary .spinner').exists()).toBe(true)
  })
})
