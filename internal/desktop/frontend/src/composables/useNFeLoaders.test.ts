import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeLoaders } from './useNFeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFePendingRow, NFeRow, NFeStatusResult } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listNFe: vi.fn(),
    statusNFe: vi.fn(),
    listNFePendingManifestacoes: vi.fn(),
  },
}))

const row = { ChaveAcesso: 'a' } as NFeRow
const statusResult = { CNPJ: '123' } as NFeStatusResult
const pendingRow = { ChaveAcesso: 'a', Kind: 'sem_ciencia' } as NFePendingRow

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useNFeLoaders', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listNFe).mockResolvedValue([row])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(statusResult)
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockResolvedValue([pendingRow])
    useNFeDocumentsStore().filter.CNPJ = '123'
  })

  it('refreshes notes, status and pendências of the selected company', async () => {
    const store = useNFeDocumentsStore()

    await useNFeLoaders().refresh('123')

    expect(store.rows).toEqual([row])
    expect(store.status).toEqual(statusResult)
    expect(store.pending).toEqual([pendingRow])
    expect(store.loading).toBe(false)
    expect(store.pendingLoading).toBe(false)
  })

  it('skips the refresh once another company is selected', async () => {
    useNFeDocumentsStore().filter.CNPJ = '456'

    await useNFeLoaders().refresh('123')

    expect(desktopClient.listNFe).not.toHaveBeenCalled()
    expect(desktopClient.statusNFe).not.toHaveBeenCalled()
    expect(desktopClient.listNFePendingManifestacoes).not.toHaveBeenCalled()
  })

  it('drops results that arrive after the company changed', async () => {
    const store = useNFeDocumentsStore()
    const call = deferred<NFeStatusResult>()
    vi.mocked(desktopClient.statusNFe).mockReturnValue(call.promise)

    const loading = useNFeLoaders().loadStatus()
    store.filter.CNPJ = '456'
    call.resolve(statusResult)

    await expect(loading).resolves.toEqual(statusResult)
    expect(store.status).toBeNull()
  })

  it('does not fail when a reload fails', async () => {
    vi.mocked(desktopClient.listNFe).mockRejectedValue(new Error('boom'))

    await expect(useNFeLoaders().refresh('123')).resolves.toBeUndefined()
    expect(useNFeDocumentsStore().pending).toEqual([pendingRow])
  })
})
