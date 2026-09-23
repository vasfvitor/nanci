import {
  daysUntil,
  formatChaveAcesso,
  formatChaveNFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatNFeNumber,
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

  it('groups a 44-character NF-e access key in blocks of four', () => {
    const chave = '35240912345678000199550010000123451123456789'
    expect(formatChaveNFe(chave)).toBe(
      '3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789'
    )
    expect(formatChaveNFe('3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789')).toBe(
      '3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789'
    )
  })

  it('returns other NF-e key lengths unchanged', () => {
    expect(formatChaveNFe('3524091234567800019955001000012345112345678')).toBe(
      '3524091234567800019955001000012345112345678'
    )
    expect(formatChaveNFe('123')).toBe('123')
    expect(formatChaveNFe('')).toBe('')
    expect(formatChaveNFe(null)).toBe('')
  })

  it('formats NF-e number and série', () => {
    expect(formatNFeNumber('12345', '1')).toBe('000.012.345 / série 1')
    expect(formatNFeNumber('12345', '001')).toBe('000.012.345 / série 1')
    expect(formatNFeNumber('123456789', '0')).toBe('123.456.789 / série 0')
    expect(formatNFeNumber('7', '')).toBe('000.000.007')
    expect(formatNFeNumber('7')).toBe('000.000.007')
    expect(formatNFeNumber('', '2')).toBe('série 2')
    expect(formatNFeNumber('', '')).toBe('')
    expect(formatNFeNumber('ABC', '1')).toBe('ABC / série 1')
  })

  it('counts whole local calendar days until a date', () => {
    const now = new Date(2026, 8, 23, 23, 30)

    expect(daysUntil(new Date(2026, 8, 23, 0, 5), now)).toBe(0)
    expect(daysUntil(new Date(2026, 8, 24, 0, 5), now)).toBe(1)
    expect(daysUntil(new Date(2026, 9, 3, 12, 0), now)).toBe(10)
    expect(daysUntil(new Date(2026, 8, 22, 23, 59), now)).toBe(-1)
    expect(daysUntil(new Date(2026, 2, 23, 12, 0), now)).toBe(-184)
    expect(daysUntil(new Date(2026, 8, 30, 12, 0).toISOString(), now)).toBe(7)
  })

  it('returns null days for empty or invalid dates', () => {
    expect(daysUntil(null)).toBeNull()
    expect(daysUntil(undefined)).toBeNull()
    expect(daysUntil('')).toBeNull()
    expect(daysUntil('not-a-date')).toBeNull()
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
