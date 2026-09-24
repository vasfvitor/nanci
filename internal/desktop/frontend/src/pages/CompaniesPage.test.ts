import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CompaniesPage from './CompaniesPage.vue'
import { desktopClient, WailsClientError } from '@/platform/wails/client'

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

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: {
    listCompanies: vi.fn(),
    listCredentials: vi.fn(),
    assignCredential: vi.fn(),
    pull: vi.fn(),
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
  UF: '',
  LastFoundNSU: null,
  SyncStartPolicy: 'from_now' as const,
  LastRunStatus: '',
  LastRunStopReason: '',
}

// Render only the credential cell slot by default, which is what most tests drive.
const credentialCellTable =
  '<div><slot v-if="rows.length" name="body-cell-credencial" v-bind="{ row: rows[0] }" /></div>'
const actionsCellTable =
  '<div><slot v-if="rows.length" name="body-cell-acoes" v-bind="{ row: rows[0] }" /></div>'

function mountPage(tableTemplate = credentialCellTable) {
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
        'q-table': {
          name: 'QTable',
          props: ['rows', 'loading'],
          template: tableTemplate,
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

describe('CompaniesPage sync errors', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  async function clickSync(error: Error) {
    vi.mocked(desktopClient.pull).mockRejectedValue(error)
    const wrapper = mountPage(actionsCellTable)
    await flushPromises()

    await wrapper.get('button[title="Sincronizar NFS-e"]').trigger('click')
    await flushPromises()
  }

  it('warns instead of failing when a sync is already running', async () => {
    await clickSync(new WailsClientError('sincronização já em andamento', 'sync_running'))

    expect(notify).toHaveBeenCalledWith({
      type: 'warning',
      message: 'Sincronização já em andamento para esta empresa.',
    })
    expect(notify).not.toHaveBeenCalledWith(expect.objectContaining({ type: 'negative' }))
  })

  it('warns when the password prompt was cancelled', async () => {
    await clickSync(new WailsClientError('operação cancelada', 'canceled'))

    expect(notify).toHaveBeenCalledWith({ type: 'warning', message: 'Sincronização cancelada.' })
  })

  it('reports other sync errors as failures', async () => {
    await clickSync(new Error('boom'))

    expect(notify).toHaveBeenCalledWith({ type: 'negative', message: 'Erro na sincronização: boom' })
  })
})
