import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import CompaniesPage from './CompaniesPage.vue'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import { deferred } from '@/test/fixtures'
import type { CompanySummary } from '@/types/desktop'

const notify = vi.fn()
const push = vi.fn()
// route is the current route; replace swaps its query as the router would.
const route = reactive<{ query: Record<string, string> }>({ query: {} })
const replace = vi.fn((to: { query: Record<string, string> }) => {
  route.query = to.query
})

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { isActive: false },
    notify,
    dialog: vi.fn(() => ({ onOk: vi.fn() })),
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push, replace }),
  useRoute: () => route,
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: {
    listCompanies: vi.fn(),
    listCredentials: vi.fn(),
    assignCredential: vi.fn(),
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
        'q-btn': { props: ['title', 'icon'], template: '<button :title="title" :data-icon="icon" />' },
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
        AddCompanyDialog: {
          name: 'AddCompanyDialog',
          props: ['modelValue'],
          emits: ['update:modelValue', 'added'],
          template: '<div />',
        },
        WelcomeCard: { name: 'WelcomeCard', emits: ['add'], template: '<div />' },
        EditCompanyDialog: { template: '<div />' },
      },
      directives: {
        ripple: {},
      },
    },
  })
}

// layoutLoad stands for the company list the layout loads when the app
// opens; the page takes the companies from the workspace store.
function layoutLoad() {
  return useWorkspaceStore().ensureCompanies()
}

describe('CompaniesPage credential assignment', () => {
  beforeEach(() => {
    localStorage.clear()
    route.query = {}
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  it('drives the table loading indicator while companies load', async () => {
    const list = deferred<CompanySummary[]>()
    vi.mocked(desktopClient.listCompanies).mockReturnValue(list.promise)

    const wrapper = mountPage()
    const loading = layoutLoad()
    await flushPromises()

    const table = wrapper.getComponent({ name: 'QTable' })
    expect(table.props('loading')).toBe(true)

    // An empty list would swap the table for the welcome card.
    list.resolve([company])
    await loading
    await flushPromises()

    expect(table.props('loading')).toBe(false)
  })

  it('rolls the select back to the stored credential when assignment fails', async () => {
    vi.mocked(desktopClient.assignCredential).mockRejectedValue(new Error('boom'))
    await layoutLoad()

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
    await layoutLoad()

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

describe('CompaniesPage initial load', () => {
  beforeEach(() => {
    localStorage.clear()
    route.query = {}
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  it('takes the companies from the layout load and lists only the credentials', async () => {
    const wrapper = mountPage()
    // The layout mounts after the page and loads the list.
    await layoutLoad()
    await flushPromises()

    expect(desktopClient.listCompanies).toHaveBeenCalledTimes(1)
    expect(desktopClient.listCredentials).toHaveBeenCalledTimes(1)
    expect(wrapper.getComponent({ name: 'QSelect' }).props('modelValue')).toBe('cred-1')
  })

  it('leaves a failed company list to the layout', async () => {
    vi.mocked(desktopClient.listCompanies).mockRejectedValue(new Error('boom'))

    const wrapper = mountPage()
    await expect(layoutLoad()).rejects.toThrow('boom')
    await flushPromises()

    expect(notify).not.toHaveBeenCalled()
    expect(useWorkspaceStore().loadError).toBe('boom')
    expect(wrapper.getComponent({ name: 'QTable' }).props('loading')).toBe(false)
  })

  it('notifies a failed credentials load once', async () => {
    vi.mocked(desktopClient.listCredentials).mockRejectedValue(new Error('boom'))
    await layoutLoad()

    const wrapper = mountPage()
    await flushPromises()

    expect(notify).toHaveBeenCalledTimes(1)
    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar credenciais: boom',
    })
    expect(wrapper.getComponent({ name: 'QSelect' }).props('modelValue')).toBe('cred-1')
  })
})

describe('CompaniesPage actions', () => {
  beforeEach(() => {
    localStorage.clear()
    route.query = {}
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  it('Ver documentos sets the workspace company and opens /documents', async () => {
    const outra = { ...company, ID: 'company-2', CNPJ: '98765432000188', Name: 'Empresa Dois' }
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company, outra])
    await layoutLoad()
    const wrapper = mountPage(actionsCellTable)
    await flushPromises()
    const workspace = useWorkspaceStore()
    workspace.cnpj = outra.CNPJ

    // The actions cell renders the first row, company.
    await wrapper.get('button[title="Ver documentos"]').trigger('click')

    expect(workspace.cnpj).toBe(company.CNPJ)
    expect(push).toHaveBeenCalledWith('/documents')
  })

  it('has no sync or reset actions', async () => {
    // Not even with debug logs on: the NFS-e page owns sync and reset.
    localStorage.setItem('nanci:logLevel', 'debug')
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    await layoutLoad()
    const wrapper = mountPage(actionsCellTable)
    await flushPromises()

    const titles = wrapper.findAll('button').map((button) => button.attributes('title'))
    expect(titles).toEqual(['Ver documentos', 'Editar'])
    expect(wrapper.find('button[data-icon="sync"]').exists()).toBe(false)
    expect(wrapper.find('button[data-icon="restart_alt"]').exists()).toBe(false)
  })
})

describe('CompaniesPage first company', () => {
  const outra = { ...company, ID: 'company-2', CNPJ: '98765432000188', Name: 'Empresa Dois' }

  beforeEach(() => {
    localStorage.clear()
    route.query = {}
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
  })

  function addDialog(wrapper: ReturnType<typeof mountPage>) {
    return wrapper.getComponent({ name: 'AddCompanyDialog' })
  }

  it('shows the welcome card with no company', async () => {
    await layoutLoad()
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'WelcomeCard' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'QTable' }).exists()).toBe(false)
  })

  it('hides it once a company exists', async () => {
    const wrapper = mountPage()
    // Before the list arrives the table shows its loading state.
    expect(wrapper.findComponent({ name: 'WelcomeCard' }).exists()).toBe(false)

    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    await layoutLoad()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'WelcomeCard' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'QTable' }).exists()).toBe(true)
  })

  it('opens the add dialog from the card', async () => {
    await layoutLoad()
    const wrapper = mountPage()
    await flushPromises()
    expect(addDialog(wrapper).props('modelValue')).toBe(false)

    wrapper.getComponent({ name: 'WelcomeCard' }).vm.$emit('add')
    await flushPromises()

    expect(addDialog(wrapper).props('modelValue')).toBe(true)
  })

  it('opens the add dialog from ?add=1 and drops the query', async () => {
    await layoutLoad()
    // Set right before the mount: the pages of earlier tests stay mounted
    // and would drop the query on the next tick.
    route.query = { add: '1' }
    const wrapper = mountPage()
    await flushPromises()

    expect(addDialog(wrapper).props('modelValue')).toBe(true)
    expect(replace).toHaveBeenCalledWith({ query: {} })
    expect(route.query).toEqual({})
  })

  it('selects the first added company and opens /documents', async () => {
    await layoutLoad()
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])

    addDialog(wrapper).vm.$emit('added', company.CNPJ)
    await flushPromises()

    expect(useWorkspaceStore().cnpj).toBe(company.CNPJ)
    expect(push).toHaveBeenCalledWith('/documents')
    expect(wrapper.findComponent({ name: 'WelcomeCard' }).exists()).toBe(false)
  })

  it('selects a later company without navigating', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    await layoutLoad()
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company, outra])

    addDialog(wrapper).vm.$emit('added', outra.CNPJ)
    await flushPromises()

    expect(useWorkspaceStore().cnpj).toBe(outra.CNPJ)
    expect(push).not.toHaveBeenCalled()
  })
})
