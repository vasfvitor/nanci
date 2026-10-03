import {
  formatCents,
  formatChaveAcesso,
  formatChaveDFe,
  formatCompetence,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatNFeNumber,
  formatParty,
  formatTime,
  normalizeText,
  parseDate,
  withViewed,
} from './formatters'

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
    expect(formatCurrencyCents(undefined)).toBe('R$ 0,00')
    expect(formatCurrencyCents(-1990)).toBe('-R$ 19,90')
  })

  it('formats integer cents without the currency symbol', () => {
    expect(formatCents(123456)).toBe('1.234,56')
    expect(formatCents(5)).toBe('0,05')
    expect(formatCents(-1990)).toBe('-19,90')
    expect(formatCents(0)).toBe('0,00')
    expect(formatCents(null)).toBe('0,00')
    expect(formatCents(undefined)).toBe('0,00')
    expect(formatCents(Number.NaN)).toBe('0,00')
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

  it('formats a competência as MM/YYYY', () => {
    expect(formatCompetence('2026-09')).toBe('09/2026')
    expect(formatCompetence('')).toBe('')
    expect(formatCompetence(null)).toBe('')
    expect(formatCompetence('2026')).toBe('2026')
  })

  it('formats local HH:MM times', () => {
    expect(formatTime(new Date(2026, 8, 23, 14, 5))).toBe('14:05')
    expect(formatTime(null, '-')).toBe('-')
    expect(formatTime('not-a-date')).toBe('')
  })
})

describe('withViewed', () => {
  it('marks the new rows among chaves and leaves the others as they are', () => {
    const now = new Date('2026-09-30T12:00:00Z')
    const before = new Date('2026-09-01T12:00:00Z')
    const rows = [
      { ChaveAcesso: 'a', ViewedAt: null },
      { ChaveAcesso: 'b', ViewedAt: before },
      { ChaveAcesso: 'c', ViewedAt: null },
    ]

    const result = withViewed(rows, ['a', 'b'], now)

    expect(result.map((row) => row.ViewedAt)).toEqual([now, before, null])
    expect(result[1]).toBe(rows[1])
    expect(result[2]).toBe(rows[2])
    expect(rows[0]?.ViewedAt).toBeNull()
  })
})
