import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useRowMap } from './useRowMap'

describe('useRowMap', () => {
  it('builds a value per row once per result set, keyed by the row', () => {
    const rows = ref([{ chave: 'a' }, { chave: 'b' }])
    const build = vi.fn((row: { chave: string }) => [row.chave.toUpperCase()])
    const map = useRowMap(rows, (row) => row.chave, build)

    const a = map.value.get('a')
    expect(a).toEqual(['A'])
    expect(map.value.get('a')).toBe(a)
    expect(build).toHaveBeenCalledTimes(2)

    rows.value = [{ chave: 'c' }]
    expect([...map.value.keys()]).toEqual(['c'])
  })
})
