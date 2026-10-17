import { shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { ConnectionTestResult } from '@/types/desktop'

// ConnectionTestSource is the service a connection test reaches: the ADN of
// the NFS-e or the SEFAZ of the CT-e.
export type ConnectionTestSource = 'nfse' | 'cte'

export const useDiagnosticsStore = defineStore('diagnostics', () => {
  const testing = shallowRef(false)
  const testResult = shallowRef<ConnectionTestResult | null>(null)
  // testSource is the service chosen for the test; testResult belongs to it.
  const testSource = shallowRef<ConnectionTestSource>('nfse')

  function clearResult() {
    testResult.value = null
  }

  return {
    testing,
    testResult,
    testSource,
    clearResult,
  }
})
