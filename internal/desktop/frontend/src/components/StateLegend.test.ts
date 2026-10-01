import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StateLegend from './StateLegend.vue'
import type { LegendSection } from '@/utils/stateLegends'

vi.mock('quasar', () => ({ useQuasar: () => ({ dark: { isActive: false } }) }))

const sections: LegendSection[] = [
  {
    title: 'Direção',
    items: [
      {
        badge: 'P',
        color: 'primary',
        name: 'Prestada',
        description: 'A empresa prestou o serviço.',
      },
    ],
  },
  {
    title: 'Prazo',
    note: 'Dias até o fim do prazo.',
    items: [{ badge: 'Vence hoje', color: 'negative', outline: true, description: 'Último dia.' }],
  },
]

function mountLegend(modelValue = true) {
  return mount(StateLegend, {
    props: { sections, modelValue },
    global: {
      stubs: {
        QExpansionItem: {
          props: ['label', 'modelValue'],
          emits: ['update:modelValue'],
          template: '<div class="expansion" :data-label="label"><slot /></div>',
        },
        QList: { template: '<div><slot /></div>' },
        QItem: { template: '<div class="item"><slot /></div>' },
        QItemSection: { template: '<div><slot /></div>' },
        QItemLabel: { template: '<div><slot /></div>' },
        QBadge: { props: ['label'], template: '<span class="badge">{{ label }}</span>' },
        QChip: { props: ['label'], template: '<span class="chip">{{ label }}</span>' },
      },
    },
  })
}

describe('StateLegend', () => {
  it('renders every section with its badges, names and descriptions', () => {
    const wrapper = mountLegend()

    expect(wrapper.find('.expansion').attributes('data-label')).toBe('Legenda dos estados')
    const rendered = wrapper
      .findAll('[data-legend-section]')
      .map((el) => el.attributes('data-legend-section'))
    expect(rendered).toEqual(['Direção', 'Prazo'])

    expect(wrapper.find('.badge').text()).toBe('P')
    expect(wrapper.text()).toContain('Prestada: A empresa prestou o serviço.')
    expect(wrapper.text()).toContain('Dias até o fim do prazo.')
  })

  it('renders outlined entries as chips', () => {
    const wrapper = mountLegend()

    expect(wrapper.findAll('.chip').map((el) => el.text())).toEqual(['Vence hoje'])
    expect(wrapper.findAll('.badge').map((el) => el.text())).toEqual(['P'])
  })

  it('renders no section while collapsed', async () => {
    const wrapper = mountLegend(false)
    expect(wrapper.find('[data-legend-section]').exists()).toBe(false)

    await wrapper.setProps({ modelValue: true })
    expect(wrapper.findAll('[data-legend-section]')).toHaveLength(2)
  })

  it('starts collapsed and follows the expansion item', async () => {
    const wrapper = mount(StateLegend, {
      props: { sections },
      global: {
        stubs: {
          QExpansionItem: {
            name: 'QExpansionItem',
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<div><slot /></div>',
          },
        },
      },
    })
    expect(wrapper.find('[data-legend-section]').exists()).toBe(false)

    await wrapper.getComponent({ name: 'QExpansionItem' }).vm.$emit('update:modelValue', true)
    expect(wrapper.find('[data-legend-section]').exists()).toBe(true)
  })
})
