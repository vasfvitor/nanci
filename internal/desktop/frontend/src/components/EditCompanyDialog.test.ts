import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EditCompanyDialog from './EditCompanyDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary } from '@/types/desktop'

vi.mock('quasar', async (importOriginal) => {
  const actual = await importOriginal<typeof import('quasar')>()
  return { ...actual, useQuasar: () => ({ notify: vi.fn(), dark: { isActive: false } }) }
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
}

function mountDialog() {
  return shallowMount(EditCompanyDialog, {
    props: { modelValue: false, companyData: company },
    global: {
      stubs: {
        'q-dialog': { template: '<div><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-separator': { template: '<div />' },
        'q-banner': { template: '<div><slot /></div>' },
        'q-btn': { name: 'QBtn', props: ['label'], template: '<button />' },
        'q-input': { template: '<div />' },
        'q-select': {
          name: 'QSelect',
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

function ufSelect(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper
    .findAllComponents({ name: 'QSelect' })
    .find((select) => select.props('label') === 'UF (opcional)')
}

async function save(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((button) => button.props('label') === 'Salvar')
    ?.trigger('click')
  await flushPromises()
}

describe('EditCompanyDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('starts from the stored UF and sends the new one', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(ufSelect(wrapper)?.props('modelValue')).toBe('SP')

    ufSelect(wrapper)?.vm.$emit('update:modelValue', 'RJ')
    await flushPromises()
    await save(wrapper)

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
    await save(wrapper)

    expect(desktopClient.updateCompany).toHaveBeenCalledWith(expect.objectContaining({ UF: '' }))
  })
})
