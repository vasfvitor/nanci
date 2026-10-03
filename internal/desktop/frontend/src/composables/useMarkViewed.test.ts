import { ref, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useMarkViewed } from './useMarkViewed'
import { deferred } from '@/test/fixtures'

type Row = { ChaveAcesso: string; ViewedAt?: Date }

// setup builds the page state the way a store holds it, so two calls of
// useMarkViewed over it stand for a page and its remount.
function setup() {
  const cnpj = ref('123')
  const filter = ref({ OnlyUnread: false })
  const rows = ref<Row[]>([{ ChaveAcesso: 'a' }, { ChaveAcesso: 'b' }])
  const filteredRows = ref<Row[]>(rows.value)
  const selected = ref<Row[]>([])
  const markingViewed = shallowRef(false)
  const mark = vi.fn(async (_cnpj: string, chaves: string[]) => chaves.length)
  const options = {
    cnpj: () => cnpj.value,
    filter,
    mark,
    rows,
    filteredRows,
    selected,
    setRows: (next: Row[]) => {
      rows.value = next
    },
    markingViewed,
  }
  return { cnpj, filter, rows, filteredRows, selected, markingViewed, mark, options }
}

describe('useMarkViewed', () => {
  it('marks by company and chaves and drops the "Novo" badge in place', async () => {
    const { rows, selected, mark, options } = setup()
    selected.value = [rows.value[0] as Row]
    const { markViewed } = useMarkViewed(options)

    await expect(markViewed(['a'])).resolves.toBe(1)

    expect(mark).toHaveBeenCalledWith('123', ['a'])
    expect(rows.value.map((row) => row.ChaveAcesso)).toEqual(['a', 'b'])
    expect(rows.value[0]?.ViewedAt).toBeInstanceOf(Date)
    expect(rows.value[1]?.ViewedAt).toBeUndefined()
    expect(selected.value).toEqual([])
  })

  it('takes the marked documents out of the list with "Somente não vistos" on', async () => {
    const { rows, mark, options } = setup()
    const { onlyUnviewed, markViewed } = useMarkViewed(options)
    onlyUnviewed.value = true

    await expect(markViewed(['a'])).resolves.toBe(1)

    expect(options.filter.value.OnlyUnread).toBe(true)
    expect(mark).toHaveBeenCalledTimes(1)
    expect(rows.value).toEqual([{ ChaveAcesso: 'b' }])
  })

  it('acts on the selection, or else on the rows the grid shows', () => {
    const { filteredRows, selected, options } = setup()
    const viewed = { ChaveAcesso: 'c', ViewedAt: new Date() }
    filteredRows.value = [{ ChaveAcesso: 'a' }, viewed]
    const { scopeRows, unviewedChaves } = useMarkViewed(options)

    expect(scopeRows.value).toEqual(filteredRows.value)
    expect(unviewedChaves.value).toEqual(['a'])

    selected.value = [viewed]
    expect(scopeRows.value).toEqual([viewed])
    expect(unviewedChaves.value).toEqual([])
  })

  it('does not mark without a company or chaves', async () => {
    const { cnpj, mark, options } = setup()
    const { markViewed } = useMarkViewed(options)

    await expect(markViewed([])).resolves.toBeNull()
    cnpj.value = ''
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
