import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFSeLoaders } from './useNFSeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useDocumentsStore } from '@/stores/documents'
import type { DocumentRow } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listDocuments: vi.fn(),
  },
}))

const row = { ChaveAcesso: 'a' } as DocumentRow

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useNFSeLoaders', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([row])
    useDocumentsStore().filter.CNPJ = '123'
  })

  it('lists the NFS-e of the selected company with the store list input', async () => {
    const store = useDocumentsStore()
    store.filter.OnlyUnread = true

    await useNFSeLoaders().search()

    expect(desktopClient.listDocuments).toHaveBeenCalledWith(store.listInput)
    expect(desktopClient.listDocuments).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: '123', OnlyUnread: true })
    )
    expect(store.documents).toEqual([row])
    expect(store.loading).toBe(false)
  })

  it('does not search without a company', async () => {
    useDocumentsStore().filter.CNPJ = ''

    await expect(useNFSeLoaders().search()).resolves.toEqual([])
    expect(desktopClient.listDocuments).not.toHaveBeenCalled()
  })

  it('drops results that arrive after the company changed', async () => {
    const store = useDocumentsStore()
    const call = deferred<DocumentRow[]>()
    vi.mocked(desktopClient.listDocuments).mockReturnValue(call.promise)

    const searching = useNFSeLoaders().search()
    expect(store.loading).toBe(true)
    store.filter.CNPJ = '456'
    call.resolve([row])

    await expect(searching).resolves.toEqual([row])
    expect(store.documents).toEqual([])
    expect(store.loading).toBe(false)
  })

  it('clears the loading flag when the list fails', async () => {
    vi.mocked(desktopClient.listDocuments).mockRejectedValue(new Error('boom'))

    await expect(useNFSeLoaders().search()).rejects.toThrow('boom')
    expect(useDocumentsStore().loading).toBe(false)
  })
})
