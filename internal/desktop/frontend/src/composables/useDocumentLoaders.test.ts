import { reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useDocumentLoaders } from './useDocumentLoaders'
import type { WorkspaceKey } from '@/stores/workspace'

type Input = { CNPJ: string; Competence: string; Code: string }
type Row = { ChaveAcesso: string }
type Status = { LastNSU: number }

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

// setup stands for a page store: the list request in listInput, whose CNPJ
// is the selected company, and the rows, loading flag, search key and status
// the loaders fill.
function setup(withStatus = true) {
  const store = reactive({
    listInput: { CNPJ: '123', Competence: '2026-05', Code: '' } as Input,
    listError: '',
    loading: false,
    rowsFor: null as WorkspaceKey | null,
    searchSeq: 0,
    status: null as Status | null,
    statusSeq: 0,
    rows: [] as Row[],
    setRows(next: Row[]) {
      store.rows = next
    },
  })
  const list = vi.fn(async (_input: Input) => [{ ChaveAcesso: 'a' }])
  const fetch = vi.fn(async (_cnpj: string) => ({ LastNSU: 7 }))

  const loaders = withStatus
    ? useDocumentLoaders(store, list, fetch)
    : useDocumentLoaders(store, list)

  return { store, list, fetch, loaders }
}

describe('useDocumentLoaders', () => {
  it('searches the selected company and stores the rows', async () => {
    const { store, list, loaders } = setup()

    await expect(loaders.search()).resolves.toEqual([{ ChaveAcesso: 'a' }])
    expect(list).toHaveBeenCalledWith({ CNPJ: '123', Competence: '2026-05', Code: '' })
    expect(store.rows).toEqual([{ ChaveAcesso: 'a' }])
    expect(store.loading).toBe(false)
  })

  it('never sends a request without a company or with a list error', async () => {
    const { store, list, loaders } = setup()

    store.listError = 'código inválido'
    await expect(loaders.search()).resolves.toEqual([])
    store.listError = ''
    store.listInput = { CNPJ: '', Competence: '', Code: '' }
    await expect(loaders.search()).resolves.toEqual([])

    expect(list).not.toHaveBeenCalled()
  })

  it('drops a list that arrives after the company changed', async () => {
    const { store, list, loaders } = setup()
    const call = deferred<Row[]>()
    list.mockReturnValue(call.promise)

    const searching = loaders.search()
    expect(store.loading).toBe(true)
    store.listInput = { ...store.listInput, CNPJ: '456' }
    call.resolve([{ ChaveAcesso: 'late' }])

    await expect(searching).resolves.toEqual([{ ChaveAcesso: 'late' }])
    expect(store.rows).toEqual([])
    expect(store.loading).toBe(false)
  })

  it('drops a list from an earlier search of the same company', async () => {
    const { store, list, loaders } = setup()
    const earlier = deferred<Row[]>()
    const later = deferred<Row[]>()
    list.mockReturnValueOnce(earlier.promise).mockReturnValueOnce(later.promise)

    const first = loaders.search()
    store.listInput = { CNPJ: '123', Competence: '2026-06', Code: '' }
    const second = loaders.search()
    later.resolve([{ ChaveAcesso: 'june' }])
    await second
    earlier.resolve([{ ChaveAcesso: 'may' }])

    await expect(first).resolves.toEqual([{ ChaveAcesso: 'may' }])
    expect(store.rows).toEqual([{ ChaveAcesso: 'june' }])
    expect(store.rowsFor).toEqual({ cnpj: '123', competence: '2026-06' })
  })

  it('keeps loading until the latest search ends', async () => {
    const { store, list, loaders } = setup()
    const earlier = deferred<Row[]>()
    const later = deferred<Row[]>()
    list.mockReturnValueOnce(earlier.promise).mockReturnValueOnce(later.promise)

    const first = loaders.search()
    const second = loaders.search()
    earlier.resolve([{ ChaveAcesso: 'old' }])
    await first
    expect(store.loading).toBe(true)
    expect(store.rows).toEqual([])

    later.resolve([{ ChaveAcesso: 'new' }])
    await second
    expect(store.loading).toBe(false)
    expect(store.rows).toEqual([{ ChaveAcesso: 'new' }])
  })

  it('records rowsFor', async () => {
    const { store, list, loaders } = setup()
    expect(store.rowsFor).toBeNull()

    await loaders.search()
    expect(store.rowsFor).toEqual({ cnpj: '123', competence: '2026-05' })

    list.mockRejectedValue(new Error('boom'))
    store.listInput = { CNPJ: '123', Competence: '', Code: '' }
    await expect(loaders.search()).rejects.toThrow('boom')
    expect(store.rowsFor).toEqual({ cnpj: '123', competence: '2026-05' })

    list.mockResolvedValue([])
    await loaders.search()
    expect(store.rowsFor).toEqual({ cnpj: '123', competence: '' })
  })

  it('clears the loading flag when the list fails', async () => {
    const { store, list, loaders } = setup()
    list.mockRejectedValue(new Error('boom'))

    await expect(loaders.search()).rejects.toThrow('boom')
    expect(store.loading).toBe(false)
  })

  it('loads the status of the selected company and drops a late one', async () => {
    const { store, fetch, loaders } = setup()

    await loaders.loadStatus()
    expect(fetch).toHaveBeenCalledWith('123')
    expect(store.status).toEqual({ LastNSU: 7 })

    const call = deferred<Status>()
    fetch.mockReturnValue(call.promise)
    const loading = loaders.loadStatus()
    store.listInput = { ...store.listInput, CNPJ: '456' }
    call.resolve({ LastNSU: 9 })
    await loading
    expect(store.status).toEqual({ LastNSU: 7 })
  })

  it('drops a status reply from an earlier load of the same company', async () => {
    const { store, list, fetch, loaders } = setup()
    const earlier = deferred<Status>()
    const later = deferred<Status>()
    fetch.mockReturnValueOnce(earlier.promise).mockReturnValueOnce(later.promise)

    const first = loaders.loadStatus()
    // A remounted page loads through its own instance of the loaders.
    const second = useDocumentLoaders(store, list, fetch).loadStatus()
    later.resolve({ LastNSU: 9 })
    await second
    earlier.resolve({ LastNSU: 8 })

    await expect(first).resolves.toEqual({ LastNSU: 8 })
    expect(store.status).toEqual({ LastNSU: 9 })
  })

  it('clears the status without a company', async () => {
    const { store, fetch, loaders } = setup()
    store.status = { LastNSU: 1 }
    store.listInput = { ...store.listInput, CNPJ: '' }

    await expect(loaders.loadStatus()).resolves.toBeNull()
    expect(store.status).toBeNull()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('has no status to load for sources without one', async () => {
    const { fetch, loaders } = setup(false)

    await expect(loaders.loadStatus('123')).resolves.toBeNull()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('refreshes only the selected company and survives failures', async () => {
    const { store, list, fetch, loaders } = setup()

    await loaders.refresh('456')
    expect(list).not.toHaveBeenCalled()
    expect(fetch).not.toHaveBeenCalled()

    list.mockRejectedValue(new Error('boom'))
    await expect(loaders.refresh('123')).resolves.toBeUndefined()
    expect(store.rows).toEqual([])
    expect(store.status).toEqual({ LastNSU: 7 })
    expect(loaders.isSelected('123')).toBe(true)
  })
})
