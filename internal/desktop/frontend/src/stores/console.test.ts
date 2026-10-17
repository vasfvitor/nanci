import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useConsoleStore } from './console'
import { desktopClient } from '@/platform/wails/client'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    setLogLevel: vi.fn(),
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

  it('rolls back the log level when the backend update fails', async () => {
    localStorage.setItem('nanci:logLevel', 'info')
    vi.mocked(desktopClient.setLogLevel).mockRejectedValue(new Error('boom'))

    const store = useConsoleStore()

    await expect(store.setLogFilter('debug')).rejects.toThrow('boom')
    expect(store.logFilterLevel).toBe('info')
    expect(localStorage.getItem('nanci:logLevel')).toBe('info')
  })
})
