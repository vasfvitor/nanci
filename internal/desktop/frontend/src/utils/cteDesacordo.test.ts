import { describe, expect, it } from 'vitest'
import { desacordoDeadlineText, desacordoMenuBlock, validateObservacao } from './cteDesacordo'
import { makeCTeRow } from '@/test/fixtures'

describe('validateObservacao', () => {
  it('accepts 15 to 255 Latin-1 characters', () => {
    expect(validateObservacao('Mercadoria não entregue')).toBeNull()
    expect(validateObservacao('Avaria: 10% da carga; ocorrência nº 12')).toBeNull()
    expect(validateObservacao('ç'.repeat(255))).toBeNull()
  })

  it('refuses what the schema refuses', () => {
    expect(validateObservacao('curta demais')).toMatch(/15 a 255/)
    expect(validateObservacao('a'.repeat(256))).toMatch(/15 a 255/)
    expect(validateObservacao(' Mercadoria não entregue')).toMatch(/espaços/)
    expect(validateObservacao('Carga “avariada” no destino')).toMatch(/aspas retas/)
    expect(validateObservacao('Carga avariada no destino 🚚')).toMatch(/emoji/)
    expect(validateObservacao('Carga avariada\nno destino')).not.toBeNull()
  })
})

describe('desacordoMenuBlock', () => {
  const tomador = makeCTeRow({
    Modelo: '57',
    CompanyRole: 'tomador',
    Papeis: ['tomador'],
    Situacao: 'autorizada',
  })

  it('offers the event to the tomador of an authorized CT-e or CT-e OS', () => {
    expect(desacordoMenuBlock(tomador)).toBeNull()
    expect(desacordoMenuBlock({ ...tomador, Modelo: '67' })).toBeNull()
    expect(
      desacordoMenuBlock({
        ...tomador,
        CompanyRole: 'destinatario',
        Papeis: ['destinatario', 'tomador'],
      })
    ).toBeNull()
  })

  it('says why it cannot be sent', () => {
    expect(desacordoMenuBlock({ ...tomador, Modelo: '64' })).toBe('Só para CT-e e CT-e OS')
    expect(
      desacordoMenuBlock({ ...tomador, CompanyRole: 'remetente', Papeis: ['remetente'] })
    ).toBe('Só o tomador pode enviar')
    expect(desacordoMenuBlock({ ...tomador, Situacao: 'cancelada' })).toBe('CT-e cancelada')
  })
})

describe('desacordoDeadlineText', () => {
  it('reads the days left to the 45-day deadline', () => {
    expect(desacordoDeadlineText(12)).toBe('faltam 12 dia(s)')
    expect(desacordoDeadlineText(0)).toBe('vence hoje')
    expect(desacordoDeadlineText(-3)).toBe('vencido há 3 dia(s)')
    expect(desacordoDeadlineText(null)).toBe('sem data de autorização')
  })
})
