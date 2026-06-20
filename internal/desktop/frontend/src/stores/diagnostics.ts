import { shallowRef } from 'vue'
import { defineStore } from 'pinia'
import type { ConnectionTestResult } from '@/types/desktop'

export const useDiagnosticsStore = defineStore('diagnostics', () => {
  const testing = shallowRef(false)
  const testResult = shallowRef<ConnectionTestResult | null>(null)

  function clearResult() {
    testResult.value = null
  }

  return {
    testing,
    testResult,
    clearResult,
  }
})
