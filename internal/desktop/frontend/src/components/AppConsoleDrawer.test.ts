import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AppConsoleDrawer from './AppConsoleDrawer.vue'
import { useConsoleStore } from '@/stores/console'
import type { LogEntry } from '@/stores/console'

const setScrollPercentage = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { isActive: false },
    notify: vi.fn(),
  }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    setLogLevel: vi.fn(),
  },
}))

function logEntry(index: number): LogEntry {
  return {
    time: '2026-07-26T12:00:00.000Z',
    level: 'INFO',
    msg: `entry ${index}`,
    attrs: '',
    raw: `entry ${index}`,
  }
}

function mountDrawer() {
  return shallowMount(AppConsoleDrawer, {
    props: { modelValue: true },
    global: {
      stubs: {
        'q-drawer': { template: '<div><slot /></div>' },
        'q-toolbar': { template: '<div><slot /></div>' },
        'q-toolbar-title': { template: '<div><slot /></div>' },
        'q-select': { template: '<div />' },
        'q-btn': { template: '<button><slot /></button>' },
        'q-tooltip': { template: '<div />' },
        'q-scroll-area': {
          template: '<div><slot /></div>',
          methods: { setScrollPercentage },
        },
      },
    },
  })
}

describe('AppConsoleDrawer', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('follows new log entries once the entry cap is reached', async () => {
    const store = useConsoleStore()
    for (let index = 0; index < 1000; index++) {
      store.pushLogEntry(logEntry(index))
    }

    mountDrawer()
    await flushPromises()
    setScrollPercentage.mockClear()

    // The store caps entries, so the list length no longer changes here.
    store.pushLogEntry(logEntry(1000))
    await flushPromises()

    expect(store.logEntries).toHaveLength(1000)
    expect(setScrollPercentage).toHaveBeenCalledWith('vertical', 1.0)
  })

  it('follows new log entries below the cap', async () => {
    const store = useConsoleStore()

    mountDrawer()
    await flushPromises()
    setScrollPercentage.mockClear()

    store.pushLogEntry(logEntry(0))
    await flushPromises()

    expect(setScrollPercentage).toHaveBeenCalledWith('vertical', 1.0)
  })
})
