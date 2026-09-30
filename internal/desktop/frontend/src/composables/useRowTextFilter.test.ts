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
    })

    local.value = rows.value
    expect(filteredRows.value.map((row) => row.chave)).toEqual(['b2'])
  })

  it('calls onChange when the text changes', async () => {
    const filterText = ref('')
    const onChange = vi.fn()
    useRowTextFilter({ rows, filterText, fields: (row) => [row.name], onChange })

    filterText.value = 'x'
    await nextTick()

    expect(onChange).toHaveBeenCalledTimes(1)
  })
})
