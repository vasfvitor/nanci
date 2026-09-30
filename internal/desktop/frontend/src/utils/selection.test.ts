import { describe, expect, it } from 'vitest'
import { pruneSelection } from './selection'

type Row = { ChaveAcesso: string; Numero: string }

describe('pruneSelection', () => {
  it('keeps the selected rows still present, swapped for their fresh rows', () => {
    const selected: Row[] = [
      { ChaveAcesso: 'b', Numero: 'old b' },
      { ChaveAcesso: 'gone', Numero: 'gone' },
      { ChaveAcesso: 'a', Numero: 'old a' },
    ]
    const next: Row[] = [
      { ChaveAcesso: 'a', Numero: 'new a' },
      { ChaveAcesso: 'b', Numero: 'new b' },
      { ChaveAcesso: 'c', Numero: 'new c' },
    ]

    expect(pruneSelection(next, selected)).toEqual([
      { ChaveAcesso: 'b', Numero: 'new b' },
      { ChaveAcesso: 'a', Numero: 'new a' },
    ])
  })

  it('returns an empty selection as it is', () => {
    const selected: Row[] = []
    expect(pruneSelection([{ ChaveAcesso: 'a', Numero: '1' }], selected)).toBe(selected)
  })
})
