import type { Ref } from 'vue'

// useExportGuard runs one export at a time for the company the grid shows.
// exporting lives in the page store, so a remounted page sees the export in
// flight.
export function useExportGuard(exporting: Ref<boolean>, cnpj: () => string) {
  // runExport calls fn with the company and returns its result, or null
  // without a company or while another export runs.
  async function runExport<T>(fn: (cnpj: string) => Promise<T>): Promise<T | null> {
    const companyCNPJ = cnpj()
    if (!companyCNPJ || exporting.value) return null
    exporting.value = true
    try {
      return await fn(companyCNPJ)
    } finally {
      exporting.value = false
    }
  }

  return { runExport }
}
