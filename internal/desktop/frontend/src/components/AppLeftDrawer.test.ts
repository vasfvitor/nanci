import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AppLeftDrawer from './AppLeftDrawer.vue'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CompanySummary } from '@/types/desktop'

const listCompanies = vi.fn()

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: (...args: unknown[]) => listCompanies(...args),
  },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

function company(CNPJ: string, Name: string): CompanySummary {
  return {
    ID: CNPJ,
    CNPJ,
    CNPJRoot: CNPJ.slice(0, 8),
    Name,
    CredentialID: '',
    CredentialLabel: '',
    CredentialCertPath: '',
    Environment: 'producao',
    UF: 'SP',
    LastFoundNSU: null,
    SyncStartPolicy: 'all',
    LastRunStatus: '',
    LastRunStopReason: '',
  }
}

const acme = company('11222333000181', 'ACME Comércio')
const wayne = company('44555666000199', 'Wayne Empreendimentos')

const stubs = {
  'q-drawer': { template: '<aside><slot /></aside>' },
  'q-list': { template: '<nav><slot /></nav>' },
  'q-item': {
    props: ['to'],
    template: '<a class="nav-link" :data-to="to"><slot /></a>',
  },
  'q-item-section': { template: '<div><slot /></div>' },
  'q-item-label': { template: '<span><slot /></span>' },
  'q-icon': { template: '<i />' },
  'q-separator': { template: '<hr />' },
  'q-btn': {
    props: ['label', 'to'],
    emits: ['click'],
    template: '<button :data-to="to" @click="$emit(\'click\')">{{ label }}</button>',
  },
  QSelect: {
    name: 'QSelect',
    props: ['modelValue', 'options', 'label', 'displayValue', 'hint', 'loading'],
    emits: ['update:modelValue'],
    template: '<div class="company-select">{{ displayValue }} {{ hint }}</div>',
  },
  CompetencePicker: {
    name: 'CompetencePicker',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<input class="competence" />',
  },
}

function mountDrawer() {
  return shallowMount(AppLeftDrawer, {
    props: { modelValue: true },
    global: { stubs, renderStubDefaultSlot: true },
  })
}

describe('AppLeftDrawer', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    listCompanies.mockReset()
  })

  it('lists the workspace companies under its labels', async () => {
    listCompanies.mockResolvedValue([acme, wayne])
    await useWorkspaceStore().loadCompanies()

    const wrapper = mountDrawer()
    const section = wrapper.get('section')
    expect(section.attributes('aria-label')).toBe('Empresa e competência')
    expect(section.attributes('title')).toContain('na NFS-e é a competência da nota (compNFSe)')

    const select = wrapper.getComponent({ name: 'QSelect' })
    expect(select.props('label')).toBe('Empresa')
    expect(select.props('modelValue')).toBe(acme.CNPJ)
    expect(select.props('displayValue')).toBe('ACME Comércio')
    expect(select.props('hint')).toBe('11.222.333/0001-81')
    expect(select.props('loading')).toBe(false)
    expect(select.props('options')).toEqual([
      expect.objectContaining({ value: acme.CNPJ, name: 'ACME Comércio', caption: '11.222.333/0001-81' }),
      expect.objectContaining({ value: wayne.CNPJ, name: 'Wayne Empreendimentos', caption: '44.555.666/0001-99' }),
    ])
  })

  it('sets the workspace company when an option is chosen', async () => {
    listCompanies.mockResolvedValue([acme, wayne])
    const workspace = useWorkspaceStore()
    await workspace.loadCompanies()

    const wrapper = mountDrawer()
    wrapper.getComponent({ name: 'QSelect' }).vm.$emit('update:modelValue', wayne.CNPJ)

    expect(workspace.cnpj).toBe(wayne.CNPJ)
  })

  it('writes the competência only when complete, and clearing writes an empty one', async () => {
    const workspace = useWorkspaceStore()
    workspace.competence = '2026-08'
    const wrapper = mountDrawer()
    const picker = wrapper.getComponent({ name: 'CompetencePicker' })
    expect(picker.props('modelValue')).toBe('2026-08')

    picker.vm.$emit('update:modelValue', '2026-0')
    await flushPromises()
    expect(workspace.competence).toBe('2026-08')

    picker.vm.$emit('update:modelValue', '2026-09')
    await flushPromises()
    expect(workspace.competence).toBe('2026-09')

    picker.vm.$emit('update:modelValue', null)
    await flushPromises()
    expect(workspace.competence).toBe('')
  })

  it('follows a competência set elsewhere', async () => {
    const workspace = useWorkspaceStore()
    const wrapper = mountDrawer()

    workspace.competence = '2026-07'
    await flushPromises()

    expect(wrapper.getComponent({ name: 'CompetencePicker' }).props('modelValue')).toBe('2026-07')
  })

  it('shows the loading select until the list arrives', () => {
    const select = mountDrawer().getComponent({ name: 'QSelect' })
    expect(select.props('loading')).toBe(true)
  })

  it('points to Empresas when no company is registered', async () => {
    listCompanies.mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()

    const wrapper = mountDrawer()

    expect(wrapper.findComponent({ name: 'QSelect' }).exists()).toBe(false)
    expect(wrapper.text()).toContain('Nenhuma empresa cadastrada.')
    const register = wrapper.findAll('button').find((item) => item.text() === 'Cadastrar empresa')
    expect(register?.attributes('data-to')).toBe('/')
  })

  it('shows a failed load and tries again', async () => {
    listCompanies.mockRejectedValueOnce(new Error('banco travado'))
    const workspace = useWorkspaceStore()
    await expect(workspace.loadCompanies()).rejects.toThrow('banco travado')

    const wrapper = mountDrawer()
    expect(wrapper.text()).toContain('Não foi possível carregar as empresas.')
    expect(wrapper.text()).toContain('banco travado')
    expect(wrapper.findComponent({ name: 'QSelect' }).exists()).toBe(false)

    listCompanies.mockResolvedValue([acme])
    const retry = wrapper.findAll('button').find((item) => item.text() === 'Tentar de novo')
    await retry?.trigger('click')
    await flushPromises()

    expect(listCompanies).toHaveBeenCalledTimes(2)
    expect(workspace.cnpj).toBe(acme.CNPJ)
    expect(wrapper.text()).not.toContain('Não foi possível carregar as empresas.')
  })

  it('links the seven pages', () => {
    const links = mountDrawer()
      .findAll('a.nav-link')
      .map((link) => [link.attributes('data-to'), link.text()])

    expect(links).toEqual([
      ['/', 'Empresas'],
      ['/documents', 'NFS-e'],
      ['/nfe', 'NF-e'],
      ['/cte', 'CT-e'],
      ['/credentials', 'Credenciais'],
      ['/query', 'Consulta Direta API'],
      ['/settings', 'Configurações'],
    ])
  })
})
