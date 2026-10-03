import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useExportGuard } from './useExportGuard'
import { deferred } from '@/test/fixtures'

describe('useExportGuard', () => {
  it('runs the export for the company and clears the flag', async () => {
    const exporting = ref(false)
    const { runExport } = useExportGuard(exporting, () => '123')
    const fn = vi.fn(async (cnpj: string) => `exported ${cnpj}`)

    await expect(runExport(fn)).resolves.toBe('exported 123')
    expect(exporting.value).toBe(false)
  })

  it('runs one export at a time, also for a second instance', async () => {
    const exporting = ref(false)
    const call = deferred<string>()
    const first = useExportGuard(exporting, () => '123').runExport(() => call.promise)
    const fn = vi.fn(async () => 'second')

    expect(exporting.value).toBe(true)
    await expect(useExportGuard(exporting, () => '123').runExport(fn)).resolves.toBeNull()
    expect(fn).not.toHaveBeenCalled()

    call.resolve('first')
    await expect(first).resolves.toBe('first')
    expect(exporting.value).toBe(false)
  })

  it('does nothing without a company and clears the flag on failure', async () => {
    const exporting = ref(false)
    const fn = vi.fn(async () => 'x')
    await expect(useExportGuard(exporting, () => '').runExport(fn)).resolves.toBeNull()
    expect(fn).not.toHaveBeenCalled()

    const failing = useExportGuard(exporting, () => '123').runExport(async () => {
      throw new Error('disco cheio')
    })
    await expect(failing).rejects.toThrow('disco cheio')
    expect(exporting.value).toBe(false)
  })
})
