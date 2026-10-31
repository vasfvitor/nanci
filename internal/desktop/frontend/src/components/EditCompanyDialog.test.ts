import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DraftBanner from './DraftBanner.vue'
import EditCompanyDialog from './EditCompanyDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import { dialogDirectives, dialogStubs, submitForm } from '@/test/dialogStubs'
import type { CompanySummary } from '@/types/desktop'

const notify = vi.hoisted(() => vi.fn())

vi.mock('quasar', async (importOriginal) => {
  const actual = await importOriginal<typeof import('quasar')>()
  return { ...actual, useQuasar: () => ({ notify, dark: { isActive: false } }) }
})

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    updateCompany: vi.fn(),
  },
}))

const company: CompanySummary = {
  ID: 'company-1',
  CNPJ: '11222333000181',
  CNPJRoot: '11222333',
  Name: 'Empresa Um',
  CredentialID: 'cred-1',
  CredentialLabel: 'Certificado A',
  CredentialCertPath: 'a.pfx',
  Environment: 'producao',
  UF: 'SP',
  LastFoundNSU: null,
  SyncStartPolicy: 'from_now',
  LastRunStatus: '',
  LastRunStopReason: '',
  LastRunError: '',
}

function mountDialog() {
  return shallowMount(EditCompanyDialog, {
    props: { modelValue: false, companyData: company },
    global: { stubs: dialogStubs, directives: dialogDirectives },
  })
}

function ufSelect(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper
    .findAllComponents({ name: 'QSelect' })
    .find((select) => select.props('label') === 'UF (opcional)')
}

function nameInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper
    .findAllComponents({ name: 'QInput' })
    .find((input) => input.props('label') === 'Nome / Razão Social')
}

// reopen closes the dialog, swaps the edited company when given, and opens it
// again.
async function reopen(wrapper: ReturnType<typeof mountDialog>, companyData?: CompanySummary) {
  await wrapper.setProps({ modelValue: false })
  if (companyData) await wrapper.setProps({ companyData })
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
}

describe('EditCompanyDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('starts from the stored UF and sends the new one', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(ufSelect(wrapper)?.props('modelValue')).toBe('SP')

    ufSelect(wrapper)?.vm.$emit('update:modelValue', 'RJ')
    await flushPromises()
    await submitForm(wrapper)

    expect(desktopClient.updateCompany).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: '11222333000181', UF: 'RJ' })
    )
  })

  it('sends an empty UF after the select is cleared', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    ufSelect(wrapper)?.vm.$emit('update:modelValue', null)
    await flushPromises()
    await submitForm(wrapper)

    expect(desktopClient.updateCompany).toHaveBeenCalledWith(expect.objectContaining({ UF: '' }))
  })

  it('keeps edits as a draft per company', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    nameInput(wrapper)?.vm.$emit('update:modelValue', 'Empresa Editada')
    await reopen(wrapper)
    expect(nameInput(wrapper)?.props('modelValue')).toBe('Empresa Editada')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)

    // Another company opens from its stored values.
    await reopen(wrapper, { ...company, CNPJ: '12345678000195', Name: 'Empresa Dois' })
    expect(nameInput(wrapper)?.props('modelValue')).toBe('Empresa Dois')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)

    await reopen(wrapper, company)
    expect(nameInput(wrapper)?.props('modelValue')).toBe('Empresa Editada')
  })

  it('drops the draft when the company changed', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    nameInput(wrapper)?.vm.$emit('update:modelValue', 'Empresa Editada')
    await flushPromises()

    // The company was renamed elsewhere while the dialog was closed.
    await reopen(wrapper, { ...company, Name: 'Empresa Renomeada' })

    expect(nameInput(wrapper)?.props('modelValue')).toBe('Empresa Renomeada')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)
  })

  it('shows an empty name inline', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    expect(nameInput(wrapper)?.props('autofocus')).toBe(true)

    nameInput(wrapper)?.vm.$emit('update:modelValue', '')
    await flushPromises()
    expect(nameInput(wrapper)?.props('error')).toBe(false)
    await submitForm(wrapper)

    expect(desktopClient.updateCompany).not.toHaveBeenCalled()
    expect(notify).not.toHaveBeenCalled()
    expect(nameInput(wrapper)?.props('error')).toBe(true)
    expect(nameInput(wrapper)?.props('errorMessage')).toBe('Informe o nome da empresa.')

    // A reopen starts without the error.
    await reopen(wrapper)
    expect(nameInput(wrapper)?.props('error')).toBe(false)
  })
})
