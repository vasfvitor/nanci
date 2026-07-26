import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AddCompanyDialog from './AddCompanyDialog.vue'

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCredentials: vi.fn().mockResolvedValue([]),
    selectCertificate: vi.fn(),
    addCompany: vi.fn(),
  },
}))

function mountDialog() {
  return shallowMount(AddCompanyDialog, {
    props: { modelValue: false },
    global: {
      stubs: {
        'q-dialog': { template: '<div><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-separator': { template: '<div />' },
        'q-btn': { template: '<button />' },
        'q-select': { template: '<div />' },
        'q-option-group': { template: '<div />' },
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

// The CNPJ input is the first q-input in the template.
function cnpjInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

describe('AddCompanyDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('clears an abandoned form when reopened after cancelling', async () => {
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    cnpjInput(wrapper).vm.$emit('update:modelValue', '12345678000199')
    await flushPromises()
    expect(cnpjInput(wrapper).props('modelValue')).toBe('12345678000199')

    // Cancel, then reopen.
    await wrapper.setProps({ modelValue: false })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(cnpjInput(wrapper).props('modelValue')).toBe('')
  })
})
