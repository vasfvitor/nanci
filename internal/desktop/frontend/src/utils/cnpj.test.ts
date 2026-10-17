import { describe, expect, it } from 'vitest'
import { cnpjError, compactCode, normalizeCNPJInput } from './cnpj'

describe('cnpj', () => {
  it('normalizes punctuation, case and length', () => {
    expect(compactCode('12.345.678/0001-95')).toBe('12345678000195')
    expect(compactCode(null)).toBe('')
    expect(normalizeCNPJInput('12.abc.345/01de-35')).toBe('12ABC34501DE35')
    expect(normalizeCNPJInput(' 12345678000195 99')).toBe('12345678000195')
    expect(normalizeCNPJInput(null)).toBe('')
    expect(normalizeCNPJInput(1234)).toBe('1234')
  })

  it('accepts valid numeric and alphanumeric CNPJs', () => {
    expect(cnpjError('12345678000195')).toBe('')
    expect(cnpjError('11.222.333/0001-81')).toBe('')
    expect(cnpjError('12ABC34501DE35')).toBe('')
    expect(cnpjError('12abc34501de35')).toBe('')
  })

  it('rejects wrong check digits, letters in the check digits, repeated characters and short input', () => {
    expect(cnpjError('12345678000199')).toBe('CNPJ inválido.')
    expect(cnpjError('12ABC34501DE45')).toBe('CNPJ inválido.')
    expect(cnpjError('12ABC34501DEAB')).toBe(
      'Os dois últimos caracteres do CNPJ devem ser numéricos.'
    )
    expect(cnpjError('00000000000000')).toBe('CNPJ inválido.')
    expect(cnpjError('11111111111111')).toBe('CNPJ inválido.')
    expect(cnpjError('1234567800019')).toBe('CNPJ deve ter 14 caracteres.')
    expect(cnpjError('')).toBe('Informe o CNPJ.')
    expect(cnpjError(null)).toBe('Informe o CNPJ.')
  })
})
