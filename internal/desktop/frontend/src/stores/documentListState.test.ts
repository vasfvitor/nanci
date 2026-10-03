import { describe, expect, it } from 'vitest'
import { documentListState } from './documentListState'

type Row = { ChaveAcesso: string; Status: string }

const row = (chave: string, Status = 'normal'): Row => ({ ChaveAcesso: chave, Status })

describe('documentListState', () => {
  it('starts empty with no action in flight', () => {
    const list = documentListState<Row>()

    expect(list.rows.value).toEqual([])
    expect(list.selected.value).toEqual([])
    expect(list.rowsFor.value).toBeNull()
    expect(list.filterText.value).toBe('')
    expect(list.loading.value).toBe(false)
    expect(list.exporting.value).toBe(false)
    expect(list.markingViewed.value).toBe(false)
    expect(list.resettingCNPJ.value).toBe('')
  })

  it('keeps only the selected rows still listed, as their fresh rows', () => {
    const list = documentListState<Row>()
    list.setRows([row('a'), row('b')])
    list.selected.value = [row('a'), row('b')]

    const freshA = row('a', 'cancelada')
    list.setRows([freshA, row('c')])

    expect(list.rows.value).toEqual([freshA, row('c')])
    expect(list.selected.value).toEqual([freshA])
  })

  it('clears the rows, the selection and their search key', () => {
    const list = documentListState<Row>()
    list.setRows([row('a')])
    list.selected.value = [row('a')]
    list.rowsFor.value = { cnpj: '123', competence: '2026-05' }

    list.clearRows()

    expect(list.rows.value).toEqual([])
    expect(list.selected.value).toEqual([])
    expect(list.rowsFor.value).toBeNull()
  })

  it('gives each list its own search gate', () => {
    const a = documentListState<Row>()
    const b = documentListState<Row>()

    const ticket = a.searchGate.begin()
    b.searchGate.begin()

    expect(ticket()).toBe(true)
  })
})
