import { describe, expect, it } from 'vitest'
import {
  completenessColor,
  completenessLabel,
  deadlineColor,
  manifestacaoColor,
  manifestacaoLabel,
  nfeEventColor,
  nfeEventLabel,
  nfeRoleColor,
  nfeRoleLabel,
  outcomeColor,
  outcomeLabel,
  situacaoColor,
  situacaoLabel,
} from './nfeDisplay'

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

  it('colors deadlines by days left', () => {
    expect(deadlineColor(null)).toBe('grey')
    expect(deadlineColor(-1)).toBe('negative')
    expect(deadlineColor(0)).toBe('negative')
    expect(deadlineColor(10)).toBe('negative')
    expect(deadlineColor(11)).toBe('warning')
    expect(deadlineColor(30)).toBe('warning')
    expect(deadlineColor(31)).toBe('grey')
  })
})
