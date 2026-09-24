import { describe, expect, it } from 'vitest'
import type { NFeEventResult, NFeRow } from '@/types/desktop'
import {
  cienciaBlockReason,
  conclusiveBlockReason,
  countOutcomes,
  validateJustificativa,
} from './nfeManifestacao'

function row(overrides: Partial<NFeRow> = {}): NFeRow {
  return {
    ID: 'rel-1',
    DocumentID: 'doc-1',
    ChaveAcesso: '35240912345678000199550010000123451123456789',
    Serie: '1',
    Numero: '12345',
    Protocolo: '',
    TpNF: '1',
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
    DaysLeft: null,
    TacitlyConfirmed: false,
    CienciaBlockReason: '',
    ConclusiveBlockReason: '',
    ...overrides,
  }
}

describe('block reasons', () => {
  it('reads the backend reasons, capitalized for display', () => {
    const blocked = row({
      CienciaBlockReason: 'já manifestada (ciencia)',
      ConclusiveBlockReason: 'NF-e já possui manifestação conclusiva (Confirmada)',
    })
    expect(cienciaBlockReason(blocked)).toBe('Já manifestada (ciencia)')
    expect(conclusiveBlockReason(blocked)).toBe('NF-e já possui manifestação conclusiva (Confirmada)')
  })

  it('returns null when the backend sends no reason', () => {
    expect(cienciaBlockReason(row())).toBeNull()
    expect(conclusiveBlockReason(row())).toBeNull()
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

describe('countOutcomes', () => {
  it('counts results by outcome, unknown ones as not sent', () => {
    const result = (Status: NFeEventResult['Status']): NFeEventResult => ({
      ChaveAcesso: '',
      TpEvento: '210210',
      Status,
      CStat: '',
      XMotivo: '',
      Protocolo: '',
    })
    expect(
      countOutcomes([
        result('registrada'),
        result('registrada'),
        result('ja_registrada'),
        result('rejeitada'),
        result('nao_enviada'),
        result(''),
      ])
    ).toEqual({ registrada: 2, ja_registrada: 1, rejeitada: 1, nao_enviada: 2 })
    expect(countOutcomes([])).toEqual({ registrada: 0, ja_registrada: 0, rejeitada: 0, nao_enviada: 0 })
  })
})
