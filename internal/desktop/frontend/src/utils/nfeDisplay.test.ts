import { describe, expect, it } from 'vitest'
import {
  conclusiveDeadlineLabel,
  deadlineColor,
  deadlineLabel,
  nfeCompleteness,
  nfeManifestacao,
  nfeRole,
  nfeSituacao,
  nfeStateBadges,
  nfeEventColor,
  nfeEventLabel,
  nfeNoteCount,
  nfePendingCount,
  outcomeColor,
  outcomeLabel,
  nfeRoleFilterOptions,
  showsConclusiveDeadline,
  nfeStatusLine,
  situacaoFilterOptions,
  situacaoLabel,
} from './nfeDisplay'
import type { NFeStatusResult } from '@/types/desktop'
import type { DisplayTable } from './sefazDisplay'

// expectShortUniqueAbbrs checks that every value of table has an
// abbreviation of one or two characters, unique within the table.
function expectShortUniqueAbbrs(table: DisplayTable) {
  const abbrs = table.values().map((value) => table.abbr(value))
  for (const abbr of abbrs) {
    expect(abbr).toMatch(/^.{1,2}$/u)
  }
  expect(new Set(abbrs).size).toBe(abbrs.length)
}

describe('nfeDisplay', () => {
  it('maps situação values', () => {
    expect(situacaoLabel('autorizada')).toBe('Autorizada')
    expect(situacaoLabel('denegada')).toBe('Denegada')
    expect(situacaoLabel('cancelada')).toBe('Cancelada')
    expect(nfeSituacao.color('autorizada')).toBe('positive')
    expect(nfeSituacao.color('denegada')).toBe('negative')
    expect(nfeSituacao.color('cancelada')).toBe('negative')
  })

  it('maps completeness values', () => {
    expect(nfeCompleteness.label('resumo')).toBe('Resumo')
    expect(nfeCompleteness.label('completa')).toBe('Completa')
    expect(nfeCompleteness.color('resumo')).toBe('warning')
    expect(nfeCompleteness.color('completa')).toBe('positive')
  })

  it('maps manifestação values', () => {
    expect(nfeManifestacao.label('nenhuma')).toBe('Sem manifestação')
    expect(nfeManifestacao.label('ciencia')).toBe('Ciência')
    expect(nfeManifestacao.label('confirmada')).toBe('Confirmada')
    expect(nfeManifestacao.label('desconhecida')).toBe('Desconhecida')
    expect(nfeManifestacao.label('nao_realizada')).toBe('Operação não realizada')
    expect(nfeManifestacao.color('nenhuma')).toBe('grey')
    expect(nfeManifestacao.color('ciencia')).toBe('info')
    expect(nfeManifestacao.color('confirmada')).toBe('positive')
    expect(nfeManifestacao.color('desconhecida')).toBe('negative')
    expect(nfeManifestacao.color('nao_realizada')).toBe('warning')
  })

  it('maps company role values', () => {
    expect(nfeRole.label('destinatario')).toBe('Destinatário')
    expect(nfeRole.label('emitente')).toBe('Emitente')
    expect(nfeRole.label('transportador')).toBe('Transportador')
    expect(nfeRole.label('autorizado')).toBe('Autorizado')
    expect(nfeRole.label('none')).toBe('Sem papel fiscal')
    expect(nfeRole.color('destinatario')).toBe('secondary')
    expect(nfeRole.color('emitente')).toBe('primary')
    expect(nfeRole.color('transportador')).toBe('accent')
    expect(nfeRole.color('autorizado')).toBe('info')
    expect(nfeRole.color('none')).toBe('grey')
  })

  it('maps event codes', () => {
    expect(nfeEventLabel('110111')).toBe('Cancelamento')
    expect(nfeEventLabel('110110')).toBe('Carta de Correção')
    expect(nfeEventLabel('210210')).toBe('Ciência')
    expect(nfeEventLabel('210200')).toBe('Confirmação')
    expect(nfeEventLabel('210220')).toBe('Desconhecimento')
    expect(nfeEventLabel('210240')).toBe('Operação não realizada')
    expect(nfeEventColor('110111')).toBe('negative')
    expect(nfeEventColor('110110')).toBe('info')
    expect(nfeEventColor('210210')).toBe('info')
    expect(nfeEventColor('210200')).toBe('positive')
    expect(nfeEventColor('210220')).toBe('negative')
    expect(nfeEventColor('210240')).toBe('warning')
  })

  it('maps event outcomes', () => {
    expect(outcomeLabel('registrada')).toBe('Registrada')
    expect(outcomeLabel('ja_registrada')).toBe('Já registrada')
    expect(outcomeLabel('rejeitada')).toBe('Rejeitada')
    expect(outcomeLabel('nao_enviada')).toBe('Não enviada')
    expect(outcomeColor('registrada')).toBe('positive')
    expect(outcomeColor('ja_registrada')).toBe('info')
    expect(outcomeColor('rejeitada')).toBe('negative')
    expect(outcomeColor('nao_enviada')).toBe('warning')
  })

  it('lists filter options after an empty "all" entry', () => {
    expect(situacaoFilterOptions).toEqual([
      { label: 'Todas', value: '' },
      { label: 'Autorizada', value: 'autorizada' },
      { label: 'Denegada', value: 'denegada' },
      { label: 'Cancelada', value: 'cancelada' },
    ])
    expect(nfeRoleFilterOptions[0]).toEqual({ label: 'Todos', value: '' })
    expect(nfeRoleFilterOptions).toHaveLength(6)
  })

  it('ignores inherited object keys', () => {
    expect(situacaoLabel('constructor')).toBe('constructor')
    expect(nfeSituacao.color('toString')).toBe('grey')
  })

  it('falls back for unknown and empty values', () => {
    expect(situacaoLabel('')).toBe('Desconhecido')
    expect(situacaoLabel('other')).toBe('other')
    expect(nfeSituacao.color('')).toBe('grey')
    expect(nfeCompleteness.label('')).toBe('Desconhecido')
    expect(nfeCompleteness.color('other')).toBe('grey')
    expect(nfeManifestacao.label('')).toBe('Desconhecido')
    expect(nfeManifestacao.color('other')).toBe('grey')
    expect(nfeRole.label('')).toBe('Desconhecido')
    expect(nfeRole.color('other')).toBe('grey')
    expect(nfeEventLabel('999999')).toBe('Evento 999999')
    expect(nfeEventLabel('')).toBe('Desconhecido')
    expect(nfeEventColor('999999')).toBe('grey')
    expect(outcomeLabel('')).toBe('Desconhecido')
    expect(outcomeColor('other')).toBe('grey')
  })

  it('colors conclusive deadlines by days left', () => {
    expect(deadlineColor(null, 'conclusiva')).toBe('grey')
    expect(deadlineColor(-1, 'conclusiva')).toBe('negative')
    expect(deadlineColor(0, 'conclusiva')).toBe('negative')
    expect(deadlineColor(10, 'conclusiva')).toBe('negative')
    expect(deadlineColor(11, 'conclusiva')).toBe('warning')
    expect(deadlineColor(30, 'conclusiva')).toBe('warning')
    expect(deadlineColor(31, 'conclusiva')).toBe('grey')
  })

  it('colors ciência deadlines with tighter thresholds', () => {
    expect(deadlineColor(null, 'ciencia')).toBe('grey')
    expect(deadlineColor(-1, 'ciencia')).toBe('negative')
    expect(deadlineColor(3, 'ciencia')).toBe('negative')
    expect(deadlineColor(4, 'ciencia')).toBe('warning')
    expect(deadlineColor(10, 'ciencia')).toBe('warning')
    expect(deadlineColor(11, 'ciencia')).toBe('grey')
  })

  it('describes days left', () => {
    expect(deadlineLabel(null)).toBe('Sem prazo')
    expect(deadlineLabel(-3)).toBe('Vencido há 3 d')
    expect(deadlineLabel(0)).toBe('Vence hoje')
    expect(deadlineLabel(12)).toBe('12 d restantes')
  })

  it('describes a passed conclusive deadline as a tacit confirmation', () => {
    expect(conclusiveDeadlineLabel(null)).toBe('Sem prazo')
    expect(conclusiveDeadlineLabel(-1)).toBe('Confirmada tacitamente')
    expect(conclusiveDeadlineLabel(0)).toBe('Vence hoje')
    expect(conclusiveDeadlineLabel(12)).toBe('12 d restantes')
  })

  it('counts pendências and notes from the status', () => {
    const status = {
      PendingCiencia: 2,
      PendingConclusiva: 3,
      TotalDestinatario: 4,
      TotalEmitente: 1,
      TotalOutros: 2,
    }
    expect(nfePendingCount(status)).toBe(5)
    expect(nfeNoteCount(status)).toBe(7)
    expect(nfePendingCount(null)).toBe(0)
    expect(nfeNoteCount(null)).toBe(0)
  })

  it('sums up the status in one line', () => {
    const status = {
      LastSyncAt: null,
      LastNSU: 10,
      MaxNSU: null,
      PendingCiencia: 1,
      PendingConclusiva: 1,
    } as unknown as NFeStatusResult
    expect(nfeStatusLine(status)).toBe('Última sincronização: nunca · NSU 10/— · Pendências: 2')
    expect(nfeStatusLine({ ...status, MaxNSU: 12 })).toContain('NSU 10/12')
  })

  it('shows the conclusive deadline only for notes with ciência', () => {
    const due = '2026-12-01T00:00:00Z'
    expect(showsConclusiveDeadline({ Manifestacao: 'ciencia', ConclusiveDue: due })).toBe(true)
    expect(showsConclusiveDeadline({ Manifestacao: 'ciencia', ConclusiveDue: null })).toBe(false)
    expect(showsConclusiveDeadline({ Manifestacao: 'nenhuma', ConclusiveDue: due })).toBe(false)
    expect(showsConclusiveDeadline({ Manifestacao: 'confirmada', ConclusiveDue: due })).toBe(false)
  })

  it('abbreviates every value with one or two unique characters', () => {
    for (const table of [nfeSituacao, nfeCompleteness, nfeManifestacao, nfeRole]) {
      expectShortUniqueAbbrs(table)
    }
    expect(nfeSituacao.values().map(nfeSituacao.abbr)).toEqual(['A', 'D', 'C'])
    expect(nfeCompleteness.values().map(nfeCompleteness.abbr)).toEqual(['R', 'X'])
    expect(nfeManifestacao.values().map(nfeManifestacao.abbr)).toEqual(['SM', 'CI', 'CO', 'DE', 'NR'])
    expect(nfeRole.values().map(nfeRole.abbr)).toEqual(['D', 'E', 'T', 'A', 'SP'])
  })

  it('lists the state badges as situação, completude, manifestação, papel', () => {
    const badges = nfeStateBadges({
      Situacao: 'autorizada',
      Completeness: 'resumo',
      Manifestacao: 'ciencia',
      CompanyRole: 'destinatario',
    })
    expect(badges.map((badge) => badge.kind)).toEqual(['Situação', 'Completude', 'Manifestação', 'Papel'])
    expect(badges.map((badge) => badge.abbr)).toEqual(['A', 'R', 'CI', 'D'])
    expect(badges[2]).toEqual({
      key: 'Manifestação:ciencia',
      abbr: 'CI',
      label: 'Ciência',
      color: 'info',
      kind: 'Manifestação',
    })
  })

  it('shows empty states as — in the badges', () => {
    const badges = nfeStateBadges({ Situacao: '', Completeness: '', Manifestacao: '', CompanyRole: '' })
    expect(badges.map((badge) => badge.abbr)).toEqual(['—', '—', '—', '—'])
  })
})
