import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCTeLoaders } from './useCTeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import type { CTeRow, CTeStatusResult } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
  },
}))

const row = { ChaveAcesso: 'a' } as CTeRow
const statusResult = { CNPJ: '123' } as CTeStatusResult

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useCTeLoaders', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCTe).mockResolvedValue([row])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(statusResult)
    useCTeDocumentsStore().filter.CNPJ = '123'
  })

  it('refreshes the CT-e list and status of the selected company', async () => {
    const store = useCTeDocumentsStore()

    await useCTeLoaders().refresh('123')

    expect(desktopClient.listCTe).toHaveBeenCalledWith(store.listInput)
    expect(store.rows).toEqual([row])
    expect(store.status).toEqual(statusResult)
    expect(store.loading).toBe(false)
  })

  it('skips the refresh once another company is selected', async () => {
    useCTeDocumentsStore().filter.CNPJ = '456'

    await useCTeLoaders().refresh('123')

    expect(desktopClient.listCTe).not.toHaveBeenCalled()
    expect(desktopClient.statusCTe).not.toHaveBeenCalled()
  })

  it('drops results that arrive after the company changed', async () => {
    const store = useCTeDocumentsStore()
    const call = deferred<CTeRow[]>()
    vi.mocked(desktopClient.listCTe).mockReturnValue(call.promise)

    const searching = useCTeLoaders().search()
    store.filter.CNPJ = '456'
    call.resolve([row])

    await expect(searching).resolves.toEqual([row])
    expect(store.rows).toEqual([])
  })

  it('does not send an NF-e key that is not 44 characters', async () => {
    const store = useCTeDocumentsStore()
    store.filter.NFeChave = '3526'

    await expect(useCTeLoaders().search()).resolves.toEqual([])
    expect(desktopClient.listCTe).not.toHaveBeenCalled()

    store.filter.NFeChave = '3'.repeat(44)
    await useCTeLoaders().search()
    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ NFeChave: '3'.repeat(44) })
    )
  })

  it('does not fail when a reload fails', async () => {
    vi.mocked(desktopClient.listCTe).mockRejectedValue(new Error('boom'))

    await expect(useCTeLoaders().refresh('123')).resolves.toBeUndefined()
    expect(useCTeDocumentsStore().status).toEqual(statusResult)
    expect(useCTeDocumentsStore().loading).toBe(false)
  })
})
