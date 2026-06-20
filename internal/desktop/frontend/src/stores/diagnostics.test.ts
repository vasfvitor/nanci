import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, describe, it } from 'vitest'
import { useDiagnosticsStore } from './diagnostics'
import type { ConnectionTestResult } from '@/types/desktop'

describe('diagnostics store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
})
