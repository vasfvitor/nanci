import { ref, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useDocumentLoaders } from './useDocumentLoaders'

type Input = { CNPJ: string; Code: string }
type Row = { ChaveAcesso: string }
type Status = { LastNSU: number }

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

function setup(withStatus = true) {
  const selected = ref('123')
  const input = ref<Input>({ CNPJ: '123', Code: '' })
  const rows = ref<Row[]>([])
  const loading = shallowRef(false)
  const status = shallowRef<Status | null>(null)
  const list = vi.fn(async (_input: Input) => [{ ChaveAcesso: 'a' }])
  const fetch = vi.fn(async (_cnpj: string) => ({ LastNSU: 7 }))

  const loaders = useDocumentLoaders<Input, Row, Status>({
    selectedCNPJ: () => selected.value,
    listInput: () => input.value,
    canList: (value) => value.Code !== 'bad',
    list,
    setRows: (next) => {
      rows.value = next
    },
    loading,
    ...(withStatus ? { status: { state: status, fetch } } : {}),
  })

  return { selected, input, rows, loading, status, list, fetch, loaders }
}

describe('useDocumentLoaders', () => {
  it('searches the selected company and stores the rows', async () => {
    const { rows, loading, list, loaders } = setup()

    await expect(loaders.search()).resolves.toEqual([{ ChaveAcesso: 'a' }])
    expect(list).toHaveBeenCalledWith({ CNPJ: '123', Code: '' })
    expect(rows.value).toEqual([{ ChaveAcesso: 'a' }])
    expect(loading.value).toBe(false)
  })

  it('never sends a request without a company or that canList rejects', async () => {
    const { input, list, loaders } = setup()

    input.value = { CNPJ: '123', Code: 'bad' }
    await expect(loaders.search()).resolves.toEqual([])
    input.value = { CNPJ: '', Code: '' }
    await expect(loaders.search()).resolves.toEqual([])

    expect(list).not.toHaveBeenCalled()
  })

  it('drops a list that arrives after the company changed', async () => {
    const { selected, rows, loading, list, loaders } = setup()
    const call = deferred<Row[]>()
    list.mockReturnValue(call.promise)

    const searching = loaders.search()
    expect(loading.value).toBe(true)
    selected.value = '456'
    call.resolve([{ ChaveAcesso: 'late' }])

    await expect(searching).resolves.toEqual([{ ChaveAcesso: 'late' }])
    expect(rows.value).toEqual([])
    expect(loading.value).toBe(false)
  })

  it('loads the status of the selected company and drops a late one', async () => {
    const { selected, status, fetch, loaders } = setup()

    await loaders.loadStatus()
    expect(fetch).toHaveBeenCalledWith('123')
    expect(status.value).toEqual({ LastNSU: 7 })

    const call = deferred<Status>()
    fetch.mockReturnValue(call.promise)
    const loading = loaders.loadStatus()
    selected.value = '456'
    call.resolve({ LastNSU: 9 })
    await loading
    expect(status.value).toEqual({ LastNSU: 7 })
  })

  it('clears the status without a company', async () => {
    const { selected, status, fetch, loaders } = setup()
    status.value = { LastNSU: 1 }
    selected.value = ''

    await expect(loaders.loadStatus()).resolves.toBeNull()
    expect(status.value).toBeNull()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('has no status to load for sources without one', async () => {
    const { fetch, loaders } = setup(false)

    await expect(loaders.loadStatus('123')).resolves.toBeNull()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('refreshes only the selected company and survives failures', async () => {
    const { rows, status, list, fetch, loaders } = setup()

    await loaders.refresh('456')
    expect(list).not.toHaveBeenCalled()
    expect(fetch).not.toHaveBeenCalled()

    list.mockRejectedValue(new Error('boom'))
    await expect(loaders.refresh('123')).resolves.toBeUndefined()
    expect(rows.value).toEqual([])
    expect(status.value).toEqual({ LastNSU: 7 })
    expect(loaders.isSelected('123')).toBe(true)
  })
})
