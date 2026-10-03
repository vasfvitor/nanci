import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DocumentFilterBar from './DocumentFilterBar.vue'

const stubs = {
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

function mountBar(props: Partial<BarProps> = {}, calls: string[] = []) {
  return mount(DocumentFilterBar, {
    props: {
      loading: false,
      exporting: false,
      exportDisabled: false,
      onlyUnviewed: false,
      markViewedCount: 0,
      'onUpdate:onlyUnviewed': (value: boolean) => calls.push(`update:onlyUnviewed ${value}`),
      onSearch: () => calls.push('search'),
      ...props,
    },
    slots: {
      // The source binds Enter on its own text fields through the search slot prop.
      default: `<template #default="{ search }">
        <input class="source-filter" @keyup.enter="search" />
      </template>`,
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
      .findAll('.source-filter, .toggle, .space, button')
      .map((el) => el.classes()[0] ?? el.text())
    expect(order).toEqual([
      'source-filter',
      'toggle',
      'space',
      'Buscar',
      'source-action',
      'Marcar vistos (3)',
      'Exportar',
    ])
  })

  it('updates the toggle, then searches', () => {
    const calls: string[] = []
    const wrapper = mountBar({ onlyUnviewed: false }, calls)

    wrapper.getComponent({ name: 'QToggle' }).vm.$emit('update:modelValue', true)

    expect(calls).toEqual(['update:onlyUnviewed true', 'search'])
  })

  it('enables "Marcar vistos" only with documents to mark', async () => {
    expect(button(mountBar(), 'Marcar vistos (0)').attributes('disabled')).toBeDefined()

    const wrapper = mountBar({ onlyUnviewed: true, markViewedCount: 2 })
    const markViewed = button(wrapper, 'Marcar vistos (2)')
    expect(markViewed.attributes('disabled')).toBeUndefined()
    await markViewed.trigger('click')
    expect(wrapper.emitted('markViewed')).toHaveLength(1)
  })

  it('searches on Enter in the fields that bind search', async () => {
    const calls: string[] = []
    const wrapper = mountBar({}, calls)

    await wrapper.find('.source-filter').trigger('keyup', { key: 'Enter' })
    expect(calls).toEqual(['search'])

    // Another key does not search.
    await wrapper.find('.source-filter').trigger('keyup', { key: 'a' })
    expect(calls).toEqual(['search'])
  })

  it('does not search while loading or when told not to', async () => {
    for (const props of [{ loading: true }, { searchDisabled: true }]) {
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
})
