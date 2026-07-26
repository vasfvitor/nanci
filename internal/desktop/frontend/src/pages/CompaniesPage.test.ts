import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CompaniesPage from './CompaniesPage.vue'
import { desktopClient } from '@/platform/wails/client'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { isActive: false },
    notify,
    dialog: vi.fn(() => ({ onOk: vi.fn() })),
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
    listCredentials: vi.fn(),
    assignCredential: vi.fn(),
    setLogLevel: vi.fn(),
  },
}))

const company = {
  ID: 'company-1',
  CNPJ: '12345678000199',
  CNPJRoot: '12345678',
  Name: 'Empresa Um',
  CredentialID: 'cred-1',
  CredentialLabel: 'Certificado A',
  CredentialCertPath: 'C:\\a.pfx',
  Environment: 'producao',
  LastFoundNSU: null,
  SyncStartPolicy: 'from_now' as const,
  LastRunStatus: '',
  LastRunStopReason: '',
}

function mountPage() {
  return shallowMount(CompaniesPage, {
    global: {
      stubs: {
        'q-page': { template: '<div><slot /></div>' },
        'q-btn': { template: '<button />' },
        'q-badge': { template: '<div />' },
        'q-td': { template: '<td><slot /></td>' },
        'q-select': {
          name: 'QSelect',
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        // Render only the credential cell slot, which is what this test drives.
        'q-table': {
          name: 'QTable',
          props: ['rows', 'loading'],
          template:
            '<div><slot v-if="rows.length" name="body-cell-credencial" v-bind="{ row: rows[0] }" /></div>',
        },
        AddCompanyDialog: { template: '<div />' },
        EditCompanyDialog: { template: '<div />' },
      },
      directives: {
        ripple: {},
      },
    },
  })
}

describe('CompaniesPage credential assignment', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  it('drives the table loading indicator while companies load', async () => {
    let resolveList!: (value: never[]) => void
    vi.mocked(desktopClient.listCompanies).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve
      }) as ReturnType<typeof desktopClient.listCompanies>
    )

    const wrapper = mountPage()
    await flushPromises()

    const table = wrapper.getComponent({ name: 'QTable' })
    expect(table.props('loading')).toBe(true)

    resolveList([])
    await flushPromises()

    expect(table.props('loading')).toBe(false)
  })

  it('rolls the select back to the stored credential when assignment fails', async () => {
    vi.mocked(desktopClient.assignCredential).mockRejectedValue(new Error('boom'))

    const wrapper = mountPage()
    await flushPromises()

    const select = wrapper.getComponent({ name: 'QSelect' })
    expect(select.props('modelValue')).toBe('cred-1')

    select.vm.$emit('update:modelValue', 'cred-2')
    await flushPromises()

    expect(desktopClient.assignCredential).toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'negative' })
    )
    expect(select.props('modelValue')).toBe('cred-1')
  })

  it('keeps the new credential when assignment succeeds', async () => {
    vi.mocked(desktopClient.assignCredential).mockResolvedValue(undefined)
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { ...company, CredentialID: 'cred-2' },
    ])

    const wrapper = mountPage()
    await flushPromises()

    const select = wrapper.getComponent({ name: 'QSelect' })
    select.vm.$emit('update:modelValue', 'cred-2')
    await flushPromises()

    expect(select.props('modelValue')).toBe('cred-2')
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'positive' })
    )
  })
})
