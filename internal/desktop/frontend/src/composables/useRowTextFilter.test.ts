import { nextTick, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useRowTextFilter } from './useRowTextFilter'

type Row = { chave: string; name: string; value: number }

const rows = ref<Row[]>([
  { chave: 'a1', name: 'São João Ltda', value: 432 },
  { chave: 'b2', name: 'Outra', value: 10 },
])

describe('useRowTextFilter', () => {
  it('matches any field ignoring case and accents', () => {
    const filterText = ref<string | null>('')
    const { filteredRows } = useRowTextFilter({
      rows,
      filterText,
      fields: (row) => [row.chave, row.name, row.value],
      pagination: ref({ page: 1 }),
    })

    expect(filteredRows.value).toHaveLength(2)
    filterText.value = 'SAO JOAO'
    expect(filteredRows.value.map((row) => row.chave)).toEqual(['a1'])
    filterText.value = '43'
    expect(filteredRows.value.map((row) => row.chave)).toEqual(['a1'])
    filterText.value = 'b2'
    expect(filteredRows.value.map((row) => row.chave)).toEqual(['b2'])
    filterText.value = null
    expect(filteredRows.value).toHaveLength(2)
  })

  it('follows new rows', () => {
    const local = ref<Row[]>([])
    const { filteredRows } = useRowTextFilter({
      rows: local,
      filterText: ref('outra'),
      fields: (row) => [row.name],
      pagination: ref({ page: 1 }),
    })

    local.value = rows.value
    expect(filteredRows.value.map((row) => row.chave)).toEqual(['b2'])
  })

  it('normalizes a row once while the list keeps it', () => {
    const local = ref<Row[]>([...rows.value])
    const filterText = ref('sao')
    const fields = vi.fn((row: Row) => [row.name])
    const { filteredRows } = useRowTextFilter({
      rows: local,
      filterText,
      fields,
      pagination: ref({ page: 1 }),
    })

    expect(filteredRows.value.map((row) => row.chave)).toEqual(['a1'])
    expect(fields).toHaveBeenCalledTimes(2)

    // A list that keeps one row object and replaces the other.
    local.value = [local.value[0] as Row, { chave: 'c3', name: 'São Paulo', value: 1 }]
    filterText.value = 'sa'
    expect(filteredRows.value.map((row) => row.chave)).toEqual(['a1', 'c3'])
    expect(fields).toHaveBeenCalledTimes(3)
  })

  it('goes back to the first page when the text changes', async () => {
    const filterText = ref('')
    const pagination = ref({ page: 3 })
    useRowTextFilter({ rows, filterText, fields: (row) => [row.name], pagination })

    filterText.value = 'x'
    await nextTick()

    expect(pagination.value.page).toBe(1)
  })
})
