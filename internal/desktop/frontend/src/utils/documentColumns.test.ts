import { describe, expect, it } from 'vitest'
import { DOCUMENT_COLUMN_NAMES, documentColumns } from './documentColumns'

type Row = {
  ChaveAcesso: string
  IssueDate?: string
  Numero: string
  EmitenteName: string
  TomadorName: string
  TotalValue: number
}

const row: Row = {
  ChaveAcesso: '35240912345678000199570010000123451123456789',
  IssueDate: '2024-09-15T10:00:00-03:00',
  Numero: '12345',
  EmitenteName: 'Transportadora',
  TomadorName: 'Cliente',
  TotalValue: 123456,
}

const columns = documentColumns<Row>({
  emitenteLabel: 'Emitente',
  destinatarioLabel: 'Tomador',
  numero: (item) => item.Numero,
  emitente: (item) => item.EmitenteName,
  destinatario: (item) => item.TomadorName,
  valor: (item) => item.TotalValue,
})

function column(name: string) {
  const found = columns.find((item) => item.name === name)
  if (!found) throw new Error(`column ${name} missing`)
  return found
}

function fieldValue(name: string) {
  const { field } = column(name)
  return typeof field === 'function' ? field(row) : row[field as keyof Row]
}

describe('documentColumns', () => {
  it('uses the standard names in the standard order', () => {
    expect(columns.map((item) => item.name)).toEqual([...DOCUMENT_COLUMN_NAMES])
    expect(DOCUMENT_COLUMN_NAMES).toEqual([
      'acoes',
      'issueDate',
      'numero',
      'chave',
      'emitente',
      'destinatario',
      'estados',
      'valor',
    ])
  })

  it('labels the columns, with the party labels of the source', () => {
    expect(columns.map((item) => item.label)).toEqual([
      'Ações',
      'Emissão',
      'Nº / Série',
      'Chave de acesso',
      'Emitente',
      'Tomador',
      'Estados',
      'Valor (R$)',
    ])
  })

  it('labels the number column as the source asks', () => {
    const numero = documentColumns<Row>({
      emitenteLabel: 'Prestador',
      destinatarioLabel: 'Tomador',
      numeroLabel: 'Número',
      numero: (item) => item.Numero,
      emitente: (item) => item.EmitenteName,
      destinatario: (item) => item.TomadorName,
      valor: (item) => item.TotalValue,
    }).find((item) => item.name === 'numero')
    expect(numero?.label).toBe('Número')
  })

  it('aligns and sorts the columns', () => {
    expect(Object.fromEntries(columns.map((item) => [item.name, item.align]))).toEqual({
      acoes: 'center',
      issueDate: 'left',
      numero: 'left',
      chave: 'left',
      emitente: 'left',
      destinatario: 'left',
      estados: 'center',
      valor: 'right',
    })
    expect(columns.filter((item) => item.sortable).map((item) => item.name)).toEqual([
      'issueDate',
      'emitente',
      'destinatario',
      'valor',
    ])
  })

  it('reads the fields of the row', () => {
    expect(fieldValue('issueDate')).toBe(row.IssueDate)
    expect(fieldValue('numero')).toBe('12345')
    expect(fieldValue('chave')).toBe(row.ChaveAcesso)
    expect(fieldValue('emitente')).toBe('Transportadora')
    expect(fieldValue('destinatario')).toBe('Cliente')
    expect(fieldValue('valor')).toBe(123456)
  })

  it('formats the date and the value without the currency symbol', () => {
    expect(column('issueDate').format?.(row.IssueDate, row)).toBe('15/09/2024')
    expect(column('valor').format?.(row.TotalValue, row)).toBe('1.234,56')
    expect(column('valor').classes).toBe('text-mono')
  })
})
