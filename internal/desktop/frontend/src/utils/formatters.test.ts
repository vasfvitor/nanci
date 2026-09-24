import {
  formatChaveAcesso,
  formatChaveDFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatNFeNumber,
  formatParty,
  formatTime,
  normalizeText,
  parseDate,
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

  it('normalizes text for accent- and case-insensitive search', () => {
    expect(normalizeText('  São Paulo ')).toBe('sao paulo')
    expect(normalizeText(null)).toBe('')
    expect(normalizeText(123)).toBe('123')
  })

  it('parses backend dates and rejects empty or invalid values', () => {
    const date = new Date(2026, 0, 2)
    expect(parseDate(date)).toBe(date)
    expect(parseDate('2026-01-02T03:04:05Z')?.toISOString()).toBe('2026-01-02T03:04:05.000Z')
    expect(parseDate('')).toBeNull()
    expect(parseDate(null)).toBeNull()
    expect(parseDate(undefined)).toBeNull()
    expect(parseDate('not-a-date')).toBeNull()
    expect(parseDate(new Date(Number.NaN))).toBeNull()
  })

  it('formats integer cents as BRL', () => {
    expect(formatCurrencyCents(123456).replace(/\s/u, ' ')).toBe('R$ 1.234,56')
    expect(formatCurrencyCents(undefined)).toBe('R$ 0,00')
  })

  it('formats access keys for display', () => {
    expect(formatChaveAcesso('NFS1234567890123')).toBe('...4567890123')
    expect(formatChaveAcesso('123')).toBe('123')
  })

  it('groups a 44-character NF-e or CT-e access key in blocks of four', () => {
    const chave = '35240912345678000199550010000123451123456789'
    expect(formatChaveDFe(chave)).toBe(
      '3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789'
    )
    expect(formatChaveDFe('3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789')).toBe(
      '3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789'
    )
  })

  it('returns other access key lengths unchanged', () => {
    expect(formatChaveDFe('3524091234567800019955001000012345112345678')).toBe(
      '3524091234567800019955001000012345112345678'
    )
    expect(formatChaveDFe('123')).toBe('123')
    expect(formatChaveDFe('')).toBe('')
    expect(formatChaveDFe(null)).toBe('')
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

  it('names a party by name, else by its formatted CPF/CNPJ', () => {
    expect(formatParty('Fornecedor', '12345678000199')).toBe('Fornecedor')
    expect(formatParty('', '12345678000199')).toBe('12.345.678/0001-99')
    expect(formatParty(null, null)).toBe('')
  })

  it('formats local HH:MM times', () => {
    expect(formatTime(new Date(2026, 8, 23, 14, 5))).toBe('14:05')
    expect(formatTime(null, '-')).toBe('-')
    expect(formatTime('not-a-date')).toBe('')
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
