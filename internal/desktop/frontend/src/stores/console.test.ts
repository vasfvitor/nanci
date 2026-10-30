import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useConsoleStore } from './console'
import { desktopClient } from '@/platform/wails/client'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    setLogLevel: vi.fn(),
  },
}))

type Handler = (payload: unknown) => void

const handlers: Record<string, Handler> = {}

vi.mock('@/platform/wails/events', () => ({
  onWailsEvent: (eventName: string, callback: Handler) => {
    handlers[eventName] = callback
    return vi.fn()
  },
}))

describe('console store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  function makeStorageThrow() {
    vi.spyOn(globalThis, 'localStorage', 'get').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
  }

  it('hydrates the saved log level', () => {
    localStorage.setItem('nanci:logLevel', 'trace')

    expect(useConsoleStore().logFilterLevel).toBe('trace')
  })

  it('falls back to info for an unknown saved level', () => {
    localStorage.setItem('nanci:logLevel', 'verbose')

    expect(useConsoleStore().logFilterLevel).toBe('info')
  })

  it('starts on info when the storage throws', () => {
    makeStorageThrow()

    expect(useConsoleStore().logFilterLevel).toBe('info')
  })

  it('still applies the log level when the storage throws', async () => {
    makeStorageThrow()
    const store = useConsoleStore()

    await store.setLogFilter('debug')

    expect(store.logFilterLevel).toBe('debug')
    expect(desktopClient.setLogLevel).toHaveBeenCalledWith('debug')
  })

  it('rolls back the log level when the storage throws and the backend fails', async () => {
    makeStorageThrow()
    vi.mocked(desktopClient.setLogLevel).mockRejectedValueOnce(new Error('boom'))
    const store = useConsoleStore()

    await expect(store.setLogFilter('warn')).rejects.toThrow('boom')
    expect(store.logFilterLevel).toBe('info')
  })

  it('persists the selected log level on success', async () => {
    const store = useConsoleStore()

    await store.setLogFilter('debug')

    expect(store.logFilterLevel).toBe('debug')
    expect(localStorage.getItem('nanci:logLevel')).toBe('debug')
    expect(desktopClient.setLogLevel).toHaveBeenCalledWith('debug')
  })

  it('adds backend-log records to the console', () => {
    const store = useConsoleStore()
    store.initLogListeners()

    handlers['backend-log']?.({
      time: '2026-06-18T14:30:01Z',
      level: 'INFO',
      msg: 'Sincronização concluída',
      attrs: 'cnpj=11222333000181',
      raw: 'time=2026-06-18T14:30:01Z level=INFO msg="Sincronização concluída"',
    })

    expect(store.logEntries).toEqual([
      {
        time: '2026-06-18T14:30:01Z',
        level: 'INFO',
        msg: 'Sincronização concluída',
        attrs: 'cnpj=11222333000181',
        raw: 'time=2026-06-18T14:30:01Z level=INFO msg="Sincronização concluída"',
      },
    ])
  })

  it('fills the empty fields of a backend-log record', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-18T15:00:00Z'))
    const store = useConsoleStore()
    store.initLogListeners()

    handlers['backend-log']?.({
      time: '',
      level: ' warning ',
      msg: 'Alíquota incomum',
      attrs: '',
      raw: '',
    })
    handlers['backend-log']?.({ time: '', level: '', msg: 'sem nível', attrs: '', raw: '' })
    vi.useRealTimers()

    expect(store.logEntries).toEqual([
      {
        time: '2026-06-18T15:00:00.000Z',
        level: 'WARNING',
        msg: 'Alíquota incomum',
        attrs: '',
        raw: 'Alíquota incomum',
      },
      {
        time: '2026-06-18T15:00:00.000Z',
        level: 'INFO',
        msg: 'sem nível',
        attrs: '',
        raw: 'sem nível',
      },
    ])
  })

  it('rolls back the log level when the backend update fails', async () => {
    localStorage.setItem('nanci:logLevel', 'info')
    vi.mocked(desktopClient.setLogLevel).mockRejectedValue(new Error('boom'))

    const store = useConsoleStore()

    await expect(store.setLogFilter('debug')).rejects.toThrow('boom')
    expect(store.logFilterLevel).toBe('info')
    expect(localStorage.getItem('nanci:logLevel')).toBe('info')
  })
})
