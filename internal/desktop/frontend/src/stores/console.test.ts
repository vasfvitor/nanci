import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
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
