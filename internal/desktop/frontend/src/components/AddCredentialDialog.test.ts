import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AddCredentialDialog from './AddCredentialDialog.vue'
import DraftBanner from './DraftBanner.vue'

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    selectCertificate: vi.fn(),
    addCredential: vi.fn(),
  },
}))

function mountDialog() {
  return shallowMount(AddCredentialDialog, {
    props: { modelValue: false },
    global: {
      stubs: {
        'q-dialog': { template: '<div><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-btn': { template: '<button />' },
        'q-input': {
          name: 'QInput',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
      },
      directives: {
        ClosePopup: {},
      },
    },
  })
}

// The label input is the first q-input in the template.
function labelInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

describe('AddCredentialDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('keeps an abandoned form as a draft when reopened after cancelling', async () => {
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    labelInput(wrapper).vm.$emit('update:modelValue', 'rótulo abandonado')
    await flushPromises()
    expect(labelInput(wrapper).props('modelValue')).toBe('rótulo abandonado')

    // Cancel, then reopen.
    await wrapper.setProps({ modelValue: false })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(labelInput(wrapper).props('modelValue')).toBe('rótulo abandonado')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)
  })
})
