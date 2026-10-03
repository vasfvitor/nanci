import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsPage from './SettingsPage.vue'
import { useConsoleStore } from '@/stores/console'
import { useWorkspaceStore } from '@/stores/workspace'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary } from '@/types/desktop'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { set: vi.fn(), isActive: false },
    notify,
  }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn().mockResolvedValue([]),
    getBuildInfo: vi.fn().mockResolvedValue({ version: '1.0.0', commit: 'abc', date: '2026-06-21' }),
    getDataDirectory: vi.fn().mockResolvedValue('C:\\data'),
    openDataDirectory: vi.fn(),
    openLogsDirectory: vi.fn(),
    exportLogs: vi.fn(),
    testConnection: vi.fn(),
    setLogLevel: vi.fn(),
  },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

vi.mock('@/platform/wails/runtime', () => ({
  desktopRuntime: {
    setDarkTheme: vi.fn(),
    setLightTheme: vi.fn(),
    setSystemDefaultTheme: vi.fn(),
  },
}))

describe('SettingsPage', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('notifies and keeps the toggle state rolled back when debug update fails', async () => {
    vi.mocked(desktopClient.setLogLevel).mockRejectedValue(new Error('boom'))

    const wrapper = shallowMount(SettingsPage, {
      global: {
        stubs: {
          'q-page': { template: '<div><slot /></div>' },
          'q-card': { template: '<div><slot /></div>' },
          'q-list': { template: '<div><slot /></div>' },
          'q-item': { template: '<div><slot /></div>' },
          'q-item-section': { template: '<div><slot /></div>' },
          'q-item-label': { template: '<div><slot /></div>' },
          'q-select': { template: '<div />' },
          'q-separator': { template: '<div />' },
          'q-space': { template: '<div />' },
          'q-btn': { template: '<button><slot /></button>' },
          'q-icon': { template: '<i />' },
          'q-toggle': {
            name: 'QToggle',
            props: ['modelValue', 'disable'],
            emits: ['update:model-value'],
            template: '<div data-testid="debug-toggle" />',
          },
        },
        directives: {
          ripple: {},
        },
      },
    })

    await flushPromises()

    wrapper.getComponent({ name: 'QToggle' }).vm.$emit('update:model-value', true)
    await flushPromises()

    const consoleStore = useConsoleStore()
    expect(consoleStore.debugEnabled).toBe(false)
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'negative',
        message: expect.stringContaining('Erro ao atualizar modo debug: boom'),
      })
    )
  })

  it('starts the connection test at the workspace company and leaves the workspace alone', async () => {
    const um = { CNPJ: '11111111000111', Name: 'Empresa Um' } as CompanySummary
    const dois = { CNPJ: '22222222000122', Name: 'Empresa Dois' } as CompanySummary
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um, dois])
    const workspace = useWorkspaceStore()
    await workspace.loadCompanies()
    workspace.cnpj = dois.CNPJ

    const wrapper = shallowMount(SettingsPage, {
      global: {
        stubs: {
          'q-page': { template: '<div><slot /></div>' },
          'q-card': { template: '<div><slot /></div>' },
          'q-list': { template: '<div><slot /></div>' },
          'q-item': { template: '<div><slot /></div>' },
          'q-item-section': { template: '<div><slot /></div>' },
          'q-item-label': { template: '<div><slot /></div>' },
          'q-select': {
            name: 'QSelect',
            props: ['modelValue', 'options', 'label'],
            emits: ['update:modelValue'],
            template: '<div />',
          },
          'q-separator': { template: '<div />' },
          'q-space': { template: '<div />' },
          'q-btn': { template: '<button><slot /></button>' },
          'q-icon': { template: '<i />' },
          'q-toggle': { template: '<div />' },
        },
        directives: {
          ripple: {},
        },
      },
    })
    await flushPromises()

    const select = wrapper
      .findAllComponents({ name: 'QSelect' })
      .find((item) => item.props('label') === 'Empresa')
    if (!select) throw new Error('no Empresa select')
    expect(select.props('modelValue')).toBe(dois.CNPJ)
    expect(select.props('options')).toEqual([
      { label: 'Empresa Um (11.111.111/0001-11)', value: um.CNPJ },
      { label: 'Empresa Dois (22.222.222/0001-22)', value: dois.CNPJ },
    ])
    expect(desktopClient.listCompanies).toHaveBeenCalledTimes(1)

    select.vm.$emit('update:modelValue', um.CNPJ)
    await flushPromises()
    expect(select.props('modelValue')).toBe(um.CNPJ)
    expect(workspace.cnpj).toBe(dois.CNPJ)
  })

  it('starts the connection test at a company listed after it opens', async () => {
    const um = { CNPJ: '11111111000111', Name: 'Empresa Um' } as CompanySummary
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um])

    const wrapper = shallowMount(SettingsPage, {
      global: {
        stubs: {
          'q-select': {
            name: 'QSelect',
            props: ['modelValue', 'label', 'loading'],
            template: '<div />',
          },
        },
        directives: { ripple: {} },
      },
    })
    await flushPromises()
    const select = () =>
      wrapper.findAllComponents({ name: 'QSelect' }).find((item) => item.props('label') === 'Empresa')
    expect(select()?.props('loading')).toBe(true)
    expect(desktopClient.listCompanies).not.toHaveBeenCalled()

    // The layout loads the list.
    await useWorkspaceStore().loadCompanies()
    await flushPromises()

    expect(select()?.props('loading')).toBe(false)
    expect(select()?.props('modelValue')).toBe(um.CNPJ)
  })
})
