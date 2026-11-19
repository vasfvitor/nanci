import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, describe, it, vi } from 'vitest'
import { useDiagnosticsStore } from './diagnostics'
import { desktopClient } from '@/platform/wails/client'
import type { ConnectionTestResult } from '@/types/desktop'
import { deferred } from '@/test/fixtures'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    testConnection: vi.fn(),
    testCTeConnection: vi.fn(),
  },
}))

const ok = { CertLoaded: true, EndpointReached: true } as unknown as ConnectionTestResult

describe('diagnostics store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('initializes with default values', () => {
    const store = useDiagnosticsStore()

    expect(store.testing).toBe(false)
    expect(store.testResult).toBeNull()
  })

  it('clears result', () => {
    const store = useDiagnosticsStore()
    store.testResult = {
      Success: true,
      Message: 'Connected',
    } as unknown as ConnectionTestResult

    store.clearResult()
    expect(store.testResult).toBeNull()
  })

  it('tests the chosen service and keeps the result', async () => {
    const store = useDiagnosticsStore()
    const pending = deferred<ConnectionTestResult>()
    vi.mocked(desktopClient.testCTeConnection).mockReturnValue(pending.promise)
    store.testSource = 'cte'
    store.testResult = ok

    const running = store.runConnectionTest('123')
    expect(store.testing).toBe(true)
    expect(store.testResult).toBeNull()
    await expect(store.runConnectionTest('123')).resolves.toBeUndefined()
    expect(desktopClient.testCTeConnection).toHaveBeenCalledTimes(1)

    pending.resolve(ok)
    await running
    expect(desktopClient.testCTeConnection).toHaveBeenCalledWith('123')
    expect(desktopClient.testConnection).not.toHaveBeenCalled()
    expect(store.testResult).toEqual(ok)
    expect(store.testing).toBe(false)
  })

  it('clears testing and rethrows when the test fails, and skips a missing company', async () => {
    const store = useDiagnosticsStore()
    vi.mocked(desktopClient.testConnection).mockRejectedValue(new Error('sem rede'))

    await expect(store.runConnectionTest('123')).rejects.toThrow('sem rede')
    expect(store.testing).toBe(false)

    await store.runConnectionTest('')
    expect(desktopClient.testConnection).toHaveBeenCalledTimes(1)
  })
})
