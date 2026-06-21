import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsPage from './SettingsPage.vue'
import { useConsoleStore } from '@/stores/console'
import { desktopClient } from '@/platform/wails/client'

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
        message: expect.stringContaining('Erro ao atualizar modo debug: Error: boom'),
      })
    )
  })
})
