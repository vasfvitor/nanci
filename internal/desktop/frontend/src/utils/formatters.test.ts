import {
  formatChaveAcesso,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
} from './formatters'
import {
  roleColor,
  roleLabel,
  statusColor,
  visibilityColor,
  visibilityLabel,
} from './nfseDisplay'

describe('formatters', () => {
  it('formats CNPJ and CPF values', () => {
    expect(formatCpfCnpj('12345678000199')).toBe('12.345.678/0001-99')
    expect(formatCpfCnpj('12ABC34501DE35')).toBe('12.ABC.345/01DE-35')
    expect(formatCpfCnpj('12.abc.345/01de-35')).toBe('12.ABC.345/01DE-35')
    expect(formatCpfCnpj('12ABC34501DEXX')).toBe('12ABC34501DEXX')
    expect(formatCpfCnpj('12345678901')).toBe('123.456.789-01')
    expect(formatCpfCnpj('ABC')).toBe('ABC')
  })

  it('handles invalid and empty dates', () => {
    expect(formatDate('not-a-date')).toBe('')
    expect(formatDate(null)).toBe('')
  })

  it('formats integer cents as BRL', () => {
    expect(formatCurrencyCents(123456).replace(/\s/u, ' ')).toBe('R$ 1.234,56')
    expect(formatCurrencyCents(undefined)).toBe('R$ 0,00')
  })

  it('formats access keys for display', () => {
    expect(formatChaveAcesso('NFS1234567890123')).toBe('...4567890123')
    expect(formatChaveAcesso('123')).toBe('123')
  })
})

describe('nfse display helpers', () => {
  it('maps known role, status, and visibility values', () => {
    expect(statusColor('normal')).toBe('positive')
    expect(roleLabel('intermediario')).toBe('Intermediário')
    expect(roleColor('tomada')).toBe('secondary')
    expect(visibilityLabel('same_root_only')).toBe('Mesmo raiz apenas')
    expect(visibilityColor('exact_tomador')).toBe('positive')
  })

  it('falls back for unknown values', () => {
    expect(statusColor('other')).toBe('grey')
    expect(roleLabel('other')).toBe('other')
    expect(visibilityLabel('')).toBe('Desconhecida')
  })
})
