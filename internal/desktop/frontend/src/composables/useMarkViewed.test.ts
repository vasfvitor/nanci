import { ref, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useMarkViewed } from './useMarkViewed'

type Row = { ChaveAcesso: string; ViewedAt?: Date }

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

// setup builds the page state the way a store holds it, so two calls of
// useMarkViewed over it stand for a page and its remount.
function setup() {
  const filter = ref({ CNPJ: '123', Competence: '2026-06', OnlyUnread: false })
  const rows = ref<Row[]>([{ ChaveAcesso: 'a' }, { ChaveAcesso: 'b' }])
  const selected = ref<Row[]>([])
  const markingViewed = shallowRef(false)
  const mark = vi.fn(async (_cnpj: string, chaves: string[]) => chaves.length)
  const search = vi.fn(async () => [])
  const options = {
    filter,
    mark,
    rows,
    selected,
    setRows: (next: Row[]) => {
      rows.value = next
    },
    search,
    markingViewed,
  }
  return { filter, rows, selected, markingViewed, mark, search, options }
}

describe('useMarkViewed', () => {
  it('marks by company and chaves and drops the "Novo" badge in place', async () => {
    const { rows, selected, mark, search, options } = setup()
    selected.value = [rows.value[0] as Row]
    const { markViewed } = useMarkViewed(options)

    await expect(markViewed(['a'])).resolves.toEqual({ count: 1, reloadError: null })

    expect(mark).toHaveBeenCalledWith('123', ['a'])
    expect(rows.value[0]?.ViewedAt).toBeInstanceOf(Date)
    expect(rows.value[1]?.ViewedAt).toBeUndefined()
    expect(selected.value).toEqual([])
    expect(search).not.toHaveBeenCalled()
  })

  it('searches again with "Somente não vistos" on', async () => {
    const { rows, search, options } = setup()
    const { onlyUnviewed, markViewed } = useMarkViewed(options)
    onlyUnviewed.value = true

    await expect(markViewed(['a', 'b'])).resolves.toEqual({ count: 2, reloadError: null })

    expect(options.filter.value.OnlyUnread).toBe(true)
    expect(search).toHaveBeenCalledTimes(1)
    expect(rows.value[0]?.ViewedAt).toBeUndefined()
  })

  it('keeps the count when the search after marking fails', async () => {
    const { filter, search, markingViewed, options } = setup()
    filter.value.OnlyUnread = true
    const failure = new Error('lista indisponível')
    search.mockRejectedValue(failure)
    const { markViewed } = useMarkViewed(options)

    await expect(markViewed(['a'])).resolves.toEqual({ count: 1, reloadError: failure })
    expect(markingViewed.value).toBe(false)
  })

  it('does not mark without a company or chaves', async () => {
    const { filter, mark, options } = setup()
    const { markViewed } = useMarkViewed(options)

    await expect(markViewed([])).resolves.toBeNull()
    filter.value.CNPJ = ''
    await expect(markViewed(['a'])).resolves.toBeNull()
    expect(mark).not.toHaveBeenCalled()
  })

  it('keeps the marking visible to a second instance while it is pending', async () => {
    const { mark, markingViewed, options } = setup()
    const call = deferred<number>()
    mark.mockReturnValue(call.promise)

    const marking = useMarkViewed(options).markViewed(['a'])
    const remounted = useMarkViewed(options)
    expect(markingViewed.value).toBe(true)
    await expect(remounted.markViewed(['a'])).resolves.toBeNull()
    expect(mark).toHaveBeenCalledTimes(1)

    call.resolve(1)
    await marking
    expect(markingViewed.value).toBe(false)
  })

  it('clears the marker and rethrows when the marking fails', async () => {
    const { mark, markingViewed, selected, rows, options } = setup()
    selected.value = [rows.value[0] as Row]
    mark.mockRejectedValue(new Error('boom'))

    await expect(useMarkViewed(options).markViewed(['a'])).rejects.toThrow('boom')
    expect(markingViewed.value).toBe(false)
    expect(selected.value).toHaveLength(1)
  })
})
