import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AddCompanyDialog from './AddCompanyDialog.vue'
import DraftBanner from './DraftBanner.vue'
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
        'q-dialog': {
          name: 'QDialog',
          props: ['noBackdropDismiss', 'persistent'],
          template: '<div><slot /></div>',
        },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-separator': { template: '<div />' },
        'q-btn': { name: 'QBtn', props: ['label'], template: '<button />' },
        'q-select': {
          name: 'QSelect',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
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

function credentialSelect(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper
    .findAllComponents({ name: 'QSelect' })
    .find((select) => select.props('label') === 'Credencial')
}

async function reopen(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper.setProps({ modelValue: false })
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
}

describe('AddCompanyDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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

  it('sends the selected UF, and an empty one after clearing', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-1', Label: 'Certificado A' },
    ] as never)
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const inputs = wrapper.findAllComponents({ name: 'QInput' })
    inputs[0]?.vm.$emit('update:modelValue', '12345678000199')
    inputs[1]?.vm.$emit('update:modelValue', 'Empresa Um')
    const ufSelect = wrapper
      .findAllComponents({ name: 'QSelect' })
      .find((select) => select.props('label') === 'UF (opcional)')
    expect(ufSelect?.props('modelValue')).toBe('')

    const save = () =>
      wrapper
        .findAllComponents({ name: 'QBtn' })
        .find((button) => button.props('label') === 'Salvar')
        ?.trigger('click')

    ufSelect?.vm.$emit('update:modelValue', 'SP')
    await flushPromises()
    await save()
    await flushPromises()
    expect(desktopClient.addCompany).toHaveBeenLastCalledWith(expect.objectContaining({ UF: 'SP' }))

    // Saving closes and resets the form; reopen and clear the select.
    await reopen(wrapper)
    inputs[0]?.vm.$emit('update:modelValue', '12345678000199')
    inputs[1]?.vm.$emit('update:modelValue', 'Empresa Um')
    ufSelect?.vm.$emit('update:modelValue', null)
    await flushPromises()
    await save()
    await flushPromises()
    expect(desktopClient.addCompany).toHaveBeenLastCalledWith(expect.objectContaining({ UF: '' }))
  })

  it('defaults a new company to producao', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-1', Label: 'Certificado A' },
    ] as never)
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const envSelect = wrapper
      .findAllComponents({ name: 'QSelect' })
      .find((select) => select.props('label') === 'Ambiente da Empresa')
    expect(envSelect?.props('modelValue')).toBe('producao')

    const inputs = wrapper.findAllComponents({ name: 'QInput' })
    inputs[0]?.vm.$emit('update:modelValue', '12345678000199')
    inputs[1]?.vm.$emit('update:modelValue', 'Empresa Um')
    await wrapper
      .findAllComponents({ name: 'QBtn' })
      .find((button) => button.props('label') === 'Salvar')
      ?.trigger('click')
    await flushPromises()
    expect(desktopClient.addCompany).toHaveBeenLastCalledWith(
      expect.objectContaining({ Environment: 'producao' }),
    )
  })

  it('keeps an abandoned form as a draft when reopened after cancelling', async () => {
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)

    cnpjInput(wrapper).vm.$emit('update:modelValue', '12345678000199')
    await flushPromises()
    expect(cnpjInput(wrapper).props('modelValue')).toBe('12345678000199')

    // Cancel, then reopen.
    await reopen(wrapper)

    expect(cnpjInput(wrapper).props('modelValue')).toBe('12345678000199')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)
  })

  it('Limpar starts an empty form', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    cnpjInput(wrapper).vm.$emit('update:modelValue', '12345678000199')
    await reopen(wrapper)

    wrapper.getComponent(DraftBanner).vm.$emit('clear')
    await flushPromises()

    expect(cnpjInput(wrapper).props('modelValue')).toBe('')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)
    await reopen(wrapper)
    expect(cnpjInput(wrapper).props('modelValue')).toBe('')
  })

  it('blocks backdrop dismiss only while dirty', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    const dialog = () => wrapper.getComponent({ name: 'QDialog' })
    expect(dialog().props('noBackdropDismiss')).toBe(false)

    cnpjInput(wrapper).vm.$emit('update:modelValue', '1')
    await flushPromises()
    expect(dialog().props('noBackdropDismiss')).toBe(true)

    cnpjInput(wrapper).vm.$emit('update:modelValue', '')
    await flushPromises()
    expect(dialog().props('noBackdropDismiss')).toBe(false)
  })

  it('falls back to the first credential when the drafted one is gone', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-1', Label: 'Certificado A' },
      { ID: 'cred-2', Label: 'Certificado B' },
    ] as never)
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    expect(credentialSelect(wrapper)?.props('modelValue')).toBe('cred-1')
    credentialSelect(wrapper)?.vm.$emit('update:modelValue', 'cred-2')
    cnpjInput(wrapper).vm.$emit('update:modelValue', '12345678000199')
    await flushPromises()

    // cred-2 is deleted while the dialog is closed.
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-3', Label: 'Certificado C' },
      { ID: 'cred-1', Label: 'Certificado A' },
    ] as never)
    await reopen(wrapper)

    expect(cnpjInput(wrapper).props('modelValue')).toBe('12345678000199')
    expect(credentialSelect(wrapper)?.props('modelValue')).toBe('cred-3')
  })
})
