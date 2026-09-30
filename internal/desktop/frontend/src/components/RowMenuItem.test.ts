import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RowMenuItem from './RowMenuItem.vue'

const stubs = {
  QItem: {
    props: ['disable'],
    emits: ['click'],
    template: '<div class="item" :data-disable="disable" @click="$emit(\'click\')"><slot /></div>',
  },
  QItemSection: { template: '<div><slot /></div>' },
  QItemLabel: {
    props: { caption: Boolean },
    template: '<div :class="caption ? \'caption\' : \'label\'"><slot /></div>',
  },
}

describe('RowMenuItem', () => {
  it('shows the label and emits click', async () => {
    const wrapper = mount(RowMenuItem, { props: { label: 'Exportar XML' }, global: { stubs } })

    expect(wrapper.find('.label').text()).toBe('Exportar XML')
    expect(wrapper.find('.caption').exists()).toBe(false)
    await wrapper.find('.item').trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
  })

  it('explains a disabled entry and never emits click', async () => {
    const wrapper = mount(RowMenuItem, {
      props: { label: 'Registrar ciência', caption: 'A empresa não é a destinatária', disable: true },
      global: { stubs },
    })

    expect(wrapper.find('.caption').text()).toBe('A empresa não é a destinatária')
    expect(wrapper.find('.item').attributes('data-disable')).toBe('true')
    await wrapper.find('.item').trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()
  })
})
