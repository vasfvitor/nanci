import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWorkspaceStore } from './workspace'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listCompanies: vi.fn() },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

function company(cnpj: string, name: string) {
  return { CNPJ: cnpj, Name: name, Environment: 'producao' } as CompanySummary
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

const um = company('11111111000111', 'Empresa Um')
const dois = company('22222222000122', 'Empresa Dois')

describe('workspace store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um, dois])
  })

  it('reads the saved competência and drops an invalid one', () => {
    localStorage.setItem('nanci:workspace:competence', '2026-03')
    expect(useWorkspaceStore().competence).toBe('2026-03')

    localStorage.setItem('nanci:workspace:competence', '2026-13')
    setActivePinia(createPinia())
    expect(useWorkspaceStore().competence).toBe('')
  })

  it('saves the company and the competência when they change', async () => {
    const store = useWorkspaceStore()
    await store.loadCompanies()

    store.cnpj = dois.CNPJ
    store.competence = '2026-05'
    await Promise.resolve()

    expect(localStorage.getItem('nanci:workspace:cnpj')).toBe(dois.CNPJ)
    expect(localStorage.getItem('nanci:workspace:competence')).toBe('2026-05')

    store.competence = ''
    await Promise.resolve()
    expect(localStorage.getItem('nanci:workspace:competence')).toBe('')
  })

  it('has no company until the list loads', () => {
    localStorage.setItem('nanci:workspace:cnpj', dois.CNPJ)
    const store = useWorkspaceStore()

    expect(store.cnpj).toBe('')
    expect(store.loaded).toBe(false)
    expect(store.selectedCompany).toBeNull()
  })

  it('keeps the saved company when it still exists', async () => {
    localStorage.setItem('nanci:workspace:cnpj', dois.CNPJ)
    const store = useWorkspaceStore()

    await expect(store.loadCompanies()).resolves.toEqual([um, dois])

    expect(store.cnpj).toBe(dois.CNPJ)
    expect(store.companies).toEqual([um, dois])
    expect(store.loaded).toBe(true)
  })

  it('falls back to the first company when the saved one is gone', async () => {
    localStorage.setItem('nanci:workspace:cnpj', '99999999000199')
    const store = useWorkspaceStore()

    await store.loadCompanies()

    expect(store.cnpj).toBe(um.CNPJ)
  })

  it('keeps the current company over the saved one', async () => {
    const store = useWorkspaceStore()
    await store.loadCompanies()
    store.cnpj = dois.CNPJ

    await store.loadCompanies()

    expect(store.cnpj).toBe(dois.CNPJ)
  })

  it('has no company without companies and keeps the saved one', async () => {
    localStorage.setItem('nanci:workspace:cnpj', dois.CNPJ)
    const store = useWorkspaceStore()
    await store.loadCompanies()
    expect(store.cnpj).toBe(dois.CNPJ)

    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await store.loadCompanies()
    await Promise.resolve()

    expect(store.cnpj).toBe('')
    expect(store.loaded).toBe(true)
    expect(localStorage.getItem('nanci:workspace:cnpj')).toBe(dois.CNPJ)

    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um, dois])
    await store.loadCompanies()
    expect(store.cnpj).toBe(dois.CNPJ)
  })

  it('fills loadError and rethrows when the list fails', async () => {
    const store = useWorkspaceStore()
    vi.mocked(desktopClient.listCompanies).mockRejectedValue(new Error('boom'))

    await expect(store.loadCompanies()).rejects.toThrow('boom')

    expect(store.loadError).toBe('boom')
    expect(store.loaded).toBe(false)
    expect(store.cnpj).toBe('')

    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um])
    await store.loadCompanies()
    expect(store.loadError).toBe('')
    expect(store.loaded).toBe(true)
  })

  it('keeps the list of the latest load', async () => {
    const store = useWorkspaceStore()
    const first = deferred<CompanySummary[]>()
    const second = deferred<CompanySummary[]>()
    vi.mocked(desktopClient.listCompanies)
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)

    const older = store.loadCompanies()
    const newer = store.loadCompanies()
    second.resolve([dois])
    await newer
    first.resolve([um, dois])
    await older

    expect(store.companies).toEqual([dois])
    expect(store.cnpj).toBe(dois.CNPJ)
  })

  it('ensureCompanies reuses the load in flight and the loaded list', async () => {
    const store = useWorkspaceStore()
    const call = deferred<CompanySummary[]>()
    vi.mocked(desktopClient.listCompanies).mockReturnValueOnce(call.promise)

    const loading = store.loadCompanies()
    const ensured = store.ensureCompanies()
    call.resolve([um])

    await expect(loading).resolves.toEqual([um])
    await expect(ensured).resolves.toEqual([um])
    await expect(store.ensureCompanies()).resolves.toEqual([um])
    expect(desktopClient.listCompanies).toHaveBeenCalledTimes(1)
  })

  it('ensureCompanies loads when nothing is loaded or in flight', async () => {
    const store = useWorkspaceStore()
    vi.mocked(desktopClient.listCompanies).mockRejectedValueOnce(new Error('boom'))
    await expect(store.ensureCompanies()).rejects.toThrow('boom')

    await expect(store.ensureCompanies()).resolves.toEqual([um, dois])
    expect(desktopClient.listCompanies).toHaveBeenCalledTimes(2)
  })

  it('exposes the selected company', async () => {
    const store = useWorkspaceStore()
    await store.loadCompanies()
    expect(store.selectedCompany).toEqual(um)

    store.cnpj = dois.CNPJ
    expect(store.selectedCompany).toEqual(dois)
  })
})
