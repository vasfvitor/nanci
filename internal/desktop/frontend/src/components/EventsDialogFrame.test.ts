import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import EventsDialogFrame from './EventsDialogFrame.vue'

const chave = '35260811222333000181570010000012341000012345'

function mountFrame(modelValue = true) {
  return shallowMount(EventsDialogFrame, {
    props: { modelValue, title: 'Eventos do CT-e', chaveAcesso: chave },
    slots: { default: '<table class="events-table" />' },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QDialog: {
          name: 'QDialog',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<div class="q-dialog-stub"><slot /></div>',
        },
        QBtn: { name: 'QBtn', props: ['label', 'icon', 'ariaLabel'], template: '<button />' },
      },
      directives: { closePopup: {} },
    },
  })
}

describe('EventsDialogFrame', () => {
  it('shows the title, the grouped key and the slotted table', () => {
    const wrapper = mountFrame()
    expect(wrapper.text()).toContain('Eventos do CT-e')
    expect(wrapper.text()).toContain('3526 0811 2223 3300 0181 5700 1000 0012 3410 0001 2345')
    expect(wrapper.find('.events-table').exists()).toBe(true)
    expect(wrapper.find('.events-dialog').exists()).toBe(true)
  })

  it('binds the dialog to v-model', async () => {
    const wrapper = mountFrame(false)
    const dialog = wrapper.getComponent({ name: 'QDialog' })
    expect(dialog.props('modelValue')).toBe(false)

    await wrapper.setProps({ modelValue: true })
    expect(dialog.props('modelValue')).toBe(true)

    dialog.vm.$emit('update:modelValue', false)
    expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
  })

  it('offers a "Fechar" button', () => {
    const labels = mountFrame()
      .findAllComponents({ name: 'QBtn' })
      .map((btn) => btn.props('label'))
    expect(labels).toContain('Fechar')
  })
})
