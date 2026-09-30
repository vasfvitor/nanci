import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RowActionsMenu from './RowActionsMenu.vue'

const stubs = {
  QBtn: {
    props: ['icon', 'color'],
    emits: ['click'],
    template:
      '<button :data-icon="icon" :data-color="color" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  QMenu: { template: '<div class="menu"><slot /></div>' },
  QList: { template: '<div class="list"><slot /></div>' },
}

function mountMenu(expanded: boolean) {
  return mount(RowActionsMenu, {
    props: { noun: 'do CT-e', expanded },
    slots: { default: '<div class="entry">Exportar XML</div>' },
    global: { stubs },
  })
}

describe('RowActionsMenu', () => {
  it('labels both buttons with the noun', () => {
    const [toggle, menu] = mountMenu(false).findAll('button')
    expect(toggle?.attributes('aria-label')).toBe('Ver detalhes do CT-e')
    expect(toggle?.attributes('aria-expanded')).toBe('false')
    expect(toggle?.attributes('data-icon')).toBe('expand_more')
    expect(menu?.attributes('aria-label')).toBe('Ações do CT-e')
    expect(menu?.attributes('data-icon')).toBe('more_vert')
  })

  it('toggles the expanded model', async () => {
    const wrapper = mountMenu(false)
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:expanded')).toEqual([[true]])

    const expanded = mountMenu(true)
    const toggle = expanded.find('button')
    expect(toggle.attributes('data-icon')).toBe('expand_less')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    await toggle.trigger('click')
    expect(expanded.emitted('update:expanded')).toEqual([[false]])
  })

  it('renders the entries inside the menu', () => {
    expect(mountMenu(false).find('.menu .list .entry').text()).toBe('Exportar XML')
  })
})
