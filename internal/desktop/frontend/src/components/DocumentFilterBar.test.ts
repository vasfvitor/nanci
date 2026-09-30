import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DocumentFilterBar from './DocumentFilterBar.vue'

const stubs = {
  QSelect: {
    name: 'QSelect',
    props: ['modelValue', 'options', 'disable'],
    emits: ['update:modelValue'],
    // Like Quasar's, the stub takes focus through a readonly input.
    template: '<div class="select q-select"><input class="q-select__focus-target" readonly /></div>',
  },
  CompetencePicker: {
    name: 'CompetencePicker',
    props: ['modelValue', 'disable'],
    emits: ['update:modelValue'],
    template: '<input class="competence" />',
  },
  QToggle: {
    name: 'QToggle',
    props: ['modelValue', 'label', 'disable'],
    emits: ['update:modelValue'],
    template: '<label class="toggle">{{ label }}</label>',
  },
  QBtn: {
    name: 'QBtn',
    props: ['label', 'disable', 'loading'],
    emits: ['click'],
    template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
  },
  QSpace: { template: '<span class="space" />' },
}

type BarProps = InstanceType<typeof DocumentFilterBar>['$props']

const companyOptions = [
  { label: 'Empresa Um (1)', value: '1' },
  { label: 'Empresa Dois (2)', value: '2' },
]

function mountBar(props: Partial<BarProps> = {}, calls: string[] = []) {
  return mount(DocumentFilterBar, {
    props: {
      cnpj: '1',
      competence: '',
      companyOptions,
      loading: false,
      exporting: false,
      exportDisabled: false,
      'onUpdate:cnpj': (value: string) => calls.push(`update:cnpj ${value}`),
      'onUpdate:onlyUnviewed': (value: boolean | undefined) => calls.push(`update:onlyUnviewed ${value}`),
      onCompanyChange: () => calls.push('companyChange'),
      onSearch: () => calls.push('search'),
      ...props,
    },
    slots: {
      default: '<input class="source-filter" /><div class="q-select"><input class="source-select" /></div>',
      actions: '<button class="source-action">Registrar ciência (2)</button>',
    },
    global: { stubs },
  })
}

function button(wrapper: ReturnType<typeof mountBar>, label: string) {
  const found = wrapper.findAll('button').find((item) => item.text().startsWith(label))
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

describe('DocumentFilterBar', () => {
  it('lays out the filters and actions in the standard order', () => {
    const wrapper = mountBar({ onlyUnviewed: false, markViewedCount: 3 })

    const order = wrapper
      .findAll('.select, .competence, .source-filter, .toggle, .space, button')
      .map((el) => el.classes()[0] ?? el.text())
    expect(order).toEqual([
      'select',
      'competence',
      'source-filter',
      'toggle',
      'space',
      'Buscar',
      'source-action',
      'Marcar vistos (3)',
      'Exportar',
    ])
  })

  it('updates the company, then asks for a company change', () => {
    const calls: string[] = []
    const wrapper = mountBar({}, calls)

    wrapper.getComponent({ name: 'QSelect' }).vm.$emit('update:modelValue', '2')

    expect(calls).toEqual(['update:cnpj 2', 'companyChange'])
  })

  it('updates the toggle, then searches', () => {
    const calls: string[] = []
    const wrapper = mountBar({ onlyUnviewed: false }, calls)

    wrapper.getComponent({ name: 'QToggle' }).vm.$emit('update:modelValue', true)

    expect(calls).toEqual(['update:onlyUnviewed true', 'search'])
  })

  it('hides the toggle and "Marcar vistos" without onlyUnviewed', () => {
    const wrapper = mountBar()

    expect(wrapper.find('.toggle').exists()).toBe(false)
    expect(wrapper.findAll('button').some((item) => item.text().startsWith('Marcar vistos'))).toBe(false)
  })

  it('enables "Marcar vistos" only with documents to mark', async () => {
    expect(button(mountBar({ onlyUnviewed: false }), 'Marcar vistos (0)').attributes('disabled')).toBeDefined()

    const wrapper = mountBar({ onlyUnviewed: true, markViewedCount: 2 })
    const markViewed = button(wrapper, 'Marcar vistos (2)')
    expect(markViewed.attributes('disabled')).toBeUndefined()
    await markViewed.trigger('click')
    expect(wrapper.emitted('markViewed')).toHaveLength(1)
  })

  it('searches on Enter in a text field only', async () => {
    const calls: string[] = []
    const wrapper = mountBar({}, calls)

    await wrapper.find('.source-filter').trigger('keyup', { key: 'Enter' })
    await wrapper.find('.competence').trigger('keyup', { key: 'Enter' })
    expect(calls).toEqual(['search', 'search'])

    // The focus target of a select, and any input inside one, belong to it.
    await wrapper.find('.q-select__focus-target').trigger('keyup', { key: 'Enter' })
    await wrapper.find('.source-select').trigger('keyup', { key: 'Enter' })
    expect(calls).toEqual(['search', 'search'])
  })

  it('does not search without a company, while loading or when told not to', async () => {
    for (const props of [{ cnpj: '' }, { loading: true }, { searchDisabled: true }]) {
      const calls: string[] = []
      const wrapper = mountBar(props, calls)

      expect(button(wrapper, 'Buscar').attributes('disabled')).toBeDefined()
      await wrapper.find('.source-filter').trigger('keyup', { key: 'Enter' })
      expect(calls).toEqual([])
    }
  })

  it('searches and exports from the buttons', async () => {
    const calls: string[] = []
    const wrapper = mountBar({}, calls)

    await button(wrapper, 'Buscar').trigger('click')
    await button(wrapper, 'Exportar').trigger('click')

    expect(calls).toEqual(['search'])
    expect(wrapper.emitted('export')).toHaveLength(1)
  })

  it('disables Exportar when there is nothing to export, an export runs or the list loads', () => {
    expect(button(mountBar(), 'Exportar').attributes('disabled')).toBeUndefined()
    expect(button(mountBar({ exportDisabled: true }), 'Exportar').attributes('disabled')).toBeDefined()
    expect(button(mountBar({ exporting: true }), 'Exportar').attributes('disabled')).toBeDefined()
    expect(button(mountBar({ loading: true }), 'Exportar').attributes('disabled')).toBeDefined()
  })

  it('titles the competência with the default explanation', () => {
    const picker = mountBar().getComponent({ name: 'CompetencePicker' })
    expect(picker.element.parentElement?.getAttribute('title')).toBe('Competência pelo mês de emissão')
  })
})
