import { describe, expect, it } from 'vitest'
import {
  ambienteColor,
  ambienteLabel,
  sefazAmbiente,
  badgeProps,
  blockedMessage,
  displayTable,
  idleWarning,
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

  it('abbreviates values, with ? for unknown and — for empty', () => {
    const table = displayTable({
      a: { label: 'Alfa', color: 'positive', abbr: 'A' },
      b: { label: 'Beta', color: 'negative' },
    })
    expect(table.abbr('a')).toBe('A')
    expect(table.abbr('b')).toBe('?')
    expect(table.abbr('z')).toBe('?')
    expect(table.abbr('constructor')).toBe('?')
    expect(table.abbr('')).toBe('—')
    expect(table.values()).toEqual(['a', 'b'])
  })

  it('lists the legend of the described values, in table order', () => {
    const table = displayTable({
      b: { label: 'Beta', color: 'negative', abbr: 'B', description: 'Valor beta.' },
      x: { label: 'Sem descrição', color: 'grey', abbr: 'X' },
      a: { label: 'Alfa', color: 'positive', description: 'Valor alfa.' },
    })
    expect(table.legend()).toEqual([
      { badge: 'B', color: 'negative', name: 'Beta', description: 'Valor beta.' },
      { badge: '?', color: 'positive', name: 'Alfa', description: 'Valor alfa.' },
    ])
  })

  it('builds state badges from the table', () => {
    const table = displayTable({ a: { label: 'Alfa', color: 'positive', abbr: 'A' } })
    expect(table.badge('a', 'Grupo')).toEqual({
      key: 'Grupo:a',
      abbr: 'A',
      label: 'Alfa',
      color: 'positive',
      kind: 'Grupo',
    })
    expect(table.badge('a', 'Grupo', true)).toMatchObject({ secondary: true })
    expect(table.badge('', 'Grupo')).toEqual({
      key: 'Grupo:',
      abbr: '—',
      label: 'Desconhecido',
      color: 'grey',
      kind: 'Grupo',
    })
  })

  it('darkens info badges in light mode only', () => {
    expect(badgeProps('info', false).color).toBe('light-blue-9')
    expect(badgeProps('info', true).color).toBe('info')
    expect(badgeProps('warning', false).color).toBe('warning')
  })

  it('picks a readable badge text color', () => {
    expect(badgeProps('warning', false).textColor).toBe('dark')
    expect(badgeProps('grey', false).textColor).toBe('dark')
    expect(badgeProps('info', false).textColor).toBe('white')
    expect(badgeProps('positive', false).textColor).toBe('white')
    expect(badgeProps('info', true).textColor).toBe('dark')
    expect(badgeProps('positive', true).textColor).toBe('dark')
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
    expect(sefazAmbiente('1')).toEqual({ label: 'Produção', color: 'negative' })
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

  it('warns only when the distribution is at risk of losing NSUs', () => {
    expect(idleWarning(null)).toBe('')
    expect(idleWarning({ IdleDays: 0 })).toBe('')
    expect(idleWarning({ IdleDays: 45 })).toBe('Sem consulta há 45 dias')
  })
})
