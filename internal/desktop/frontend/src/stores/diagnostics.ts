import { shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
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

  // runConnectionTest tests the company's certificate against the chosen
  // service. testing and the result live here, so they survive leaving the
  // page; a failure is thrown for the page to report.
  async function runConnectionTest(cnpj: string) {
    if (!cnpj || testing.value) return
    testing.value = true
    clearResult()
    try {
      testResult.value =
        testSource.value === 'cte'
          ? await desktopClient.testCTeConnection(cnpj)
          : await desktopClient.testConnection(cnpj)
    } finally {
      testing.value = false
    }
  }

  return {
    testing,
    testResult,
    testSource,
    clearResult,
    runConnectionTest,
  }
})
