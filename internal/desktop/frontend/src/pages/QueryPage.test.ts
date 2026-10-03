import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QueryPage from './QueryPage.vue'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { isActive: false },
    notify: vi.fn(),
  }),
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: {
    listCompanies: vi.fn(),
    listDocuments: vi.fn(),
    queryNFSeEvents: vi.fn(),
  },
}))

const acme = { ID: '1', CNPJ: '11222333000181', Name: 'ACME Comércio' }
const wayne = { ID: '2', CNPJ: '44555666000199', Name: 'Wayne Empreendimentos' }

function mountPage() {
  return shallowMount(QueryPage, {
    global: {
      mocks: { $q: { dark: { isActive: false } } },
      stubs: {
        'q-page': { template: '<div><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-form': { template: '<form><slot /></form>' },
        'q-banner': { template: '<div class="banner"><slot /><slot name="action" /></div>' },
        'q-btn': {
          props: ['label', 'to', 'disable'],
          template: '<button :data-to="to" :disabled="disable">{{ label }}</button>',
        },
        QInput: {
          name: 'QInput',
          props: { modelValue: String, label: String, hint: String, readonly: Boolean },
          template: '<input />',
        },
      },
    },
  })
}

function button(wrapper: ReturnType<typeof mountPage>, label: string) {
  const found = wrapper.findAll('button').find((item) => item.text() === label)
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

describe('QueryPage', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([])
  })

  it('shows the workspace company it authenticates as', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([acme, wayne] as never)
    const workspace = useWorkspaceStore()
    await workspace.loadCompanies()

    const wrapper = mountPage()
    await flushPromises()

    const company = wrapper.getComponent({ name: 'QInput' })
    expect(company.props('label')).toBe('Empresa (autenticação)')
    expect(company.props('hint')).toBe('Troque no menu lateral')
    expect(company.props('readonly')).toBe(true)
    expect(company.props('modelValue')).toBe('ACME Comércio (11.222.333/0001-81)')
    expect(wrapper.find('.banner').exists()).toBe(false)
    expect(button(wrapper, 'Consultar no ADN').attributes('disabled')).toBeUndefined()

    workspace.cnpj = wayne.CNPJ
    await flushPromises()

    expect(company.props('modelValue')).toBe('Wayne Empreendimentos (44.555.666/0001-99)')
  })

  it('points to Empresas and disables the query without a company', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.get('.banner').text()).toContain('Nenhuma empresa cadastrada.')
    expect(button(wrapper, 'Cadastrar empresa').attributes('data-to')).toBe('/')
    expect(button(wrapper, 'Consultar no ADN').attributes('disabled')).toBeDefined()
    expect(wrapper.getComponent({ name: 'QInput' }).props('modelValue')).toBe('')
  })
})
