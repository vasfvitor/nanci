import { describe, expect, it } from 'vitest'
import {
  ambienteColor,
  ambienteLabel,
  badgeColor,
  badgeProps,
  badgeTextColor,
  blockedMessage,
  displayTable,
} from './sefazDisplay'

describe('sefazDisplay', () => {
  it('looks values up in a display table', () => {
    const table = displayTable(
      { a: { label: 'Alfa', color: 'positive' } },
      (value) => `Valor ${value}`
    )
    expect(table.label('a')).toBe('Alfa')
    expect(table.color('a')).toBe('positive')
    expect(table.label('b')).toBe('Valor b')
    expect(table.label('')).toBe('Desconhecido')
    expect(table.color('b')).toBe('grey')
    expect(table.label('constructor')).toBe('Valor constructor')
    expect(table.options('Todos')).toEqual([
      { label: 'Todos', value: '' },
      { label: 'Alfa', value: 'a' },
    ])
  })

  it('darkens info badges in light mode only', () => {
    expect(badgeColor('info', false)).toBe('light-blue-9')
    expect(badgeColor('info', true)).toBe('info')
    expect(badgeColor('warning', false)).toBe('warning')
  })

  it('picks a readable badge text color', () => {
    expect(badgeTextColor('warning', false)).toBe('dark')
    expect(badgeTextColor('grey', false)).toBe('dark')
    expect(badgeTextColor('info', false)).toBe('white')
    expect(badgeTextColor('positive', false)).toBe('white')
    expect(badgeTextColor('info', true)).toBe('dark')
    expect(badgeTextColor('positive', true)).toBe('dark')
  })

  it('binds badge fill and text colors together', () => {
    expect(badgeProps('info', false)).toEqual({ color: 'light-blue-9', textColor: 'white' })
    expect(badgeProps('warning', true)).toEqual({ color: 'warning', textColor: 'dark' })
  })

  it('colors the ambiente by tpAmb', () => {
    expect(ambienteColor('1')).toBe('negative')
    expect(ambienteColor('2')).toBe('warning')
    expect(ambienteColor('')).toBe('grey')
  })

  it('names the environment of tpAmb', () => {
    expect(ambienteLabel('1')).toBe('Produção')
    expect(ambienteLabel('2')).toBe('Homologação')
    expect(ambienteLabel('')).toBe('Ambiente desconhecido')
  })

  it('explains each block reason', () => {
    const info = { RequestsLastHour: 20, RequestBudget: 20 }
    expect(blockedMessage({ ...info, BlockedReason: 'caught_up' }, '14:32')).toBe(
      'Sem novos documentos; próxima consulta a partir de 14:32'
    )
    expect(blockedMessage({ ...info, BlockedReason: 'consumo_indevido' }, '14:32')).toBe(
      'Consultas bloqueadas pela SEFAZ até 14:32 (cStat 656)'
    )
    expect(blockedMessage({ ...info, BlockedReason: 'rate_budget' }, '14:32')).toBe(
      'Limite de consultas por hora atingido (20/20); aguarde até 14:32'
    )
    expect(blockedMessage({ ...info, BlockedReason: '' }, '14:32')).toBe(
      'Próxima consulta permitida a partir de 14:32'
    )
  })
})
