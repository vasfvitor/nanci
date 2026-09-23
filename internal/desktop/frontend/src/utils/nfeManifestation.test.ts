import { describe, expect, it } from 'vitest'
import type { NFeRow } from '@/types/desktop'
import {
  cienciaBlockReason,
  conclusiveBlockReason,
  validateJustificativa,
} from './nfeManifestation'

function row(overrides: Partial<NFeRow> = {}): NFeRow {
  return {
    ID: 'rel-1',
    DocumentID: 'doc-1',
    ChaveAcesso: '35240912345678000199550010000123451123456789',
    Serie: '1',
    Numero: '12345',
    Protocolo: '',
    TipoOperacao: '1',
    EmitenteCNPJ: '12345678000199',
    EmitenteName: 'Fornecedor',
    EmitenteIE: '',
    DestinatarioCNPJ: '98765432000199',
    DestinatarioName: 'Empresa',
    TotalValue: 10000,
    Situacao: 'autorizada',
    Completeness: 'resumo',
    Manifestacao: 'nenhuma',
    CompanyRole: 'destinatario',
    EventCount: 0,
    ...overrides,
  }
}

describe('cienciaBlockReason', () => {
  it('allows an authorized note addressed to the company without manifestação', () => {
    expect(cienciaBlockReason(row())).toBeNull()
  })

  it('blocks when the company is not the destinatário', () => {
    for (const role of ['emitente', 'transportador', 'autorizado', 'none', ''] as const) {
      expect(cienciaBlockReason(row({ CompanyRole: role }))).toBe(
        'Somente o destinatário pode manifestar'
      )
    }
  })

  it('blocks notes that are not authorized', () => {
    expect(cienciaBlockReason(row({ Situacao: 'cancelada' }))).toBe('Nota cancelada')
    expect(cienciaBlockReason(row({ Situacao: 'denegada' }))).toBe('Nota denegada')
    expect(cienciaBlockReason(row({ Situacao: '' }))).toBe('Situação da nota desconhecida')
  })

  it('blocks notes that already have a manifestação', () => {
    for (const manifestacao of ['ciencia', 'confirmada', 'desconhecida', 'nao_realizada'] as const) {
      expect(cienciaBlockReason(row({ Manifestacao: manifestacao }))).toBe('Já possui manifestação')
    }
    expect(cienciaBlockReason(row({ Manifestacao: '' }))).toBe('Manifestação atual desconhecida')
  })
})

describe('conclusiveBlockReason', () => {
  it('allows conclusive events before and after ciência', () => {
    expect(conclusiveBlockReason(row({ Manifestacao: 'nenhuma' }))).toBeNull()
    expect(conclusiveBlockReason(row({ Manifestacao: 'ciencia' }))).toBeNull()
  })

  it('does not block on a past conclusive deadline', () => {
    const expired = row({ Manifestacao: 'ciencia', ConclusiveDue: '2020-01-01T00:00:00Z' })
    expect(conclusiveBlockReason(expired)).toBeNull()
  })

  it('blocks every conclusive event once one is registered', () => {
    for (const manifestacao of ['confirmada', 'desconhecida', 'nao_realizada'] as const) {
      expect(conclusiveBlockReason(row({ Manifestacao: manifestacao }))).toBe(
        'Já possui manifestação conclusiva'
      )
    }
  })

  it('applies the destinatário and situação rules', () => {
    expect(conclusiveBlockReason(row({ CompanyRole: 'emitente' }))).toBe(
      'Somente o destinatário pode manifestar'
    )
    expect(conclusiveBlockReason(row({ Situacao: 'cancelada' }))).toBe('Nota cancelada')
  })
})

describe('validateJustificativa', () => {
  it('requires 15 to 255 characters', () => {
    expect(validateJustificativa('a'.repeat(14))).toBe(
      'A justificativa deve ter pelo menos 15 caracteres'
    )
    expect(validateJustificativa('a'.repeat(15))).toBeNull()
    expect(validateJustificativa('a'.repeat(255))).toBeNull()
    expect(validateJustificativa('a'.repeat(256))).toBe(
      'A justificativa deve ter no máximo 255 caracteres'
    )
  })

  it('ignores surrounding whitespace', () => {
    expect(validateJustificativa(`   ${'a'.repeat(14)}   `)).toBe(
      'A justificativa deve ter pelo menos 15 caracteres'
    )
    expect(validateJustificativa(`  ${'a'.repeat(15)}\n`)).toBeNull()
    expect(validateJustificativa(`  ${'a'.repeat(255)}  `)).toBeNull()
    expect(validateJustificativa(' '.repeat(20))).toBe(
      'A justificativa deve ter pelo menos 15 caracteres'
    )
  })

  it('rejects empty values', () => {
    expect(validateJustificativa('')).toBe('A justificativa deve ter pelo menos 15 caracteres')
    expect(validateJustificativa(null)).toBe('A justificativa deve ter pelo menos 15 caracteres')
  })
})
