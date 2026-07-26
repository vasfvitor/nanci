import { flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AddCompanyDialog from './AddCompanyDialog.vue'
import { desktopClient } from '@/platform/wails/client'

// Keep Quasar's real date helpers; only useQuasar needs stubbing.
vi.mock('quasar', async (importOriginal) => {
  const actual = await importOriginal<typeof import('quasar')>()
  return { ...actual, useQuasar: () => ({ notify: vi.fn() }) }
})

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
        'q-btn': { name: 'QBtn', props: ['label'], template: '<button />' },
        'q-select': { template: '<div />' },
        'q-option-group': {
          name: 'QOptionGroup',
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
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
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('anchors a relative sync start to the local date, not the UTC one', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-1', Label: 'Certificado A' },
    ] as never)
    // Late evening in UTC-3: the UTC calendar date is already the next day.
    // Only Date is faked, so flushPromises' timers keep working.
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-07-26T23:30:00-03:00'))

    const wrapper = mountDialog()
    // Open so credentials load and the form is reset before filling it in.
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const inputs = wrapper.findAllComponents({ name: 'QInput' })
    inputs[0]?.vm.$emit('update:modelValue', '12345678000199')
    inputs[1]?.vm.$emit('update:modelValue', 'Empresa Um')
    // The first option group selects the initial-history policy.
    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', 'last_12_months')
    await flushPromises()

    const saveButton = wrapper
      .findAllComponents({ name: 'QBtn' })
      .find((button) => button.props('label') === 'Salvar')
    await saveButton?.trigger('click')
    await flushPromises()

    expect(desktopClient.addCompany).toHaveBeenCalledWith(
      expect.objectContaining({
        SyncStartPolicy: 'since_date',
        SyncStartDate: '2025-07-26',
      })
    )
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
