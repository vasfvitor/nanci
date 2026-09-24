import { describe, expect, it } from 'vitest'
import {
  ambienteColor,
  ambienteLabel,
  badgeColor,
  badgeTextColor,
  blockedMessage,
  completenessColor,
  completenessLabel,
  conclusiveDeadlineLabel,
  deadlineColor,
  deadlineLabel,
  manifestacaoColor,
  manifestacaoLabel,
  nfeEventColor,
  nfeEventLabel,
  nfeNoteCount,
  nfePendingCount,
  nfeRoleColor,
  nfeRoleLabel,
  outcomeColor,
  outcomeLabel,
  nfeRoleFilterOptions,
  nfeStatusLine,
  situacaoColor,
  situacaoFilterOptions,
  situacaoLabel,
} from './nfeDisplay'
import type { NFeStatusResult } from '@/types/desktop'

describe('nfeDisplay', () => {
  it('maps situação values', () => {
    expect(situacaoLabel('autorizada')).toBe('Autorizada')
    expect(situacaoLabel('denegada')).toBe('Denegada')
    expect(situacaoLabel('cancelada')).toBe('Cancelada')
    expect(situacaoColor('autorizada')).toBe('positive')
    expect(situacaoColor('denegada')).toBe('negative')
    expect(situacaoColor('cancelada')).toBe('negative')
  })

  it('maps completeness values', () => {
    expect(completenessLabel('resumo')).toBe('Resumo')
    expect(completenessLabel('completa')).toBe('Completa')
    expect(completenessColor('resumo')).toBe('warning')
    expect(completenessColor('completa')).toBe('positive')
  })

  it('maps manifestação values', () => {
    expect(manifestacaoLabel('nenhuma')).toBe('Sem manifestação')
    expect(manifestacaoLabel('ciencia')).toBe('Ciência')
    expect(manifestacaoLabel('confirmada')).toBe('Confirmada')
    expect(manifestacaoLabel('desconhecida')).toBe('Desconhecida')
    expect(manifestacaoLabel('nao_realizada')).toBe('Operação não realizada')
    expect(manifestacaoColor('nenhuma')).toBe('grey')
    expect(manifestacaoColor('ciencia')).toBe('info')
    expect(manifestacaoColor('confirmada')).toBe('positive')
    expect(manifestacaoColor('desconhecida')).toBe('negative')
    expect(manifestacaoColor('nao_realizada')).toBe('warning')
  })

  it('maps company role values', () => {
    expect(nfeRoleLabel('destinatario')).toBe('Destinatário')
    expect(nfeRoleLabel('emitente')).toBe('Emitente')
    expect(nfeRoleLabel('transportador')).toBe('Transportador')
    expect(nfeRoleLabel('autorizado')).toBe('Autorizado')
    expect(nfeRoleLabel('none')).toBe('Sem papel fiscal')
    expect(nfeRoleColor('destinatario')).toBe('secondary')
    expect(nfeRoleColor('emitente')).toBe('primary')
    expect(nfeRoleColor('transportador')).toBe('accent')
    expect(nfeRoleColor('autorizado')).toBe('info')
    expect(nfeRoleColor('none')).toBe('grey')
  })

  it('maps event codes', () => {
    expect(nfeEventLabel('110111')).toBe('Cancelamento')
    expect(nfeEventLabel('110110')).toBe('Carta de Correção')
    expect(nfeEventLabel('210210')).toBe('Ciência')
    expect(nfeEventLabel('210200')).toBe('Confirmação')
    expect(nfeEventLabel('210220')).toBe('Desconhecimento')
    expect(nfeEventLabel('210240')).toBe('Operação não Realizada')
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
    expect(situacaoColor('toString')).toBe('grey')
  })

  it('falls back for unknown and empty values', () => {
    expect(situacaoLabel('')).toBe('Desconhecido')
    expect(situacaoLabel('other')).toBe('other')
    expect(situacaoColor('')).toBe('grey')
    expect(completenessLabel('')).toBe('Desconhecido')
    expect(completenessColor('other')).toBe('grey')
    expect(manifestacaoLabel('')).toBe('Desconhecido')
    expect(manifestacaoColor('other')).toBe('grey')
    expect(nfeRoleLabel('')).toBe('Desconhecido')
    expect(nfeRoleColor('other')).toBe('grey')
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
})
