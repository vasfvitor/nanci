const cnpjPattern = /^([A-Z0-9]{2})([A-Z0-9]{3})([A-Z0-9]{3})([A-Z0-9]{4})(\d{2})$/

export function formatCpfCnpj(value: string | null | undefined) {
  if (!value) return ''
  // CNPJs may carry letters in the first 12 positions (IN RFB 2.229/2024);
  // the check digits are always numeric.
  const alnum = value.replace(/[^0-9A-Za-z]/g, '').toUpperCase()
  if (alnum.length === 14 && cnpjPattern.test(alnum)) {
    return alnum.replace(cnpjPattern, '$1.$2.$3/$4-$5')
  }
  const digits = value.replace(/\D/g, '')
  if (digits.length === 11) {
    return digits.replace(/^(\d{3})(\d{3})(\d{3})(\d{2})$/, '$1.$2.$3-$4')
  }
  return value
}

// normalizeText folds case and accents, so a text search matches "sao" to
// "São".
export function normalizeText(value: unknown): string {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .toLowerCase()
    .trim()
}

// parseDate turns a backend date into a Date, or null when value is empty or
// invalid.
export function parseDate(value: string | Date | null | undefined): Date | null {
  if (!value) return null
  const parsed = value instanceof Date ? value : new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

export function formatDate(value: string | Date | null | undefined) {
  return parseDate(value)?.toLocaleDateString('pt-BR') ?? ''
}

export function formatDateTime(value: string | Date | null | undefined, fallback = '-') {
  return parseDate(value)?.toLocaleString('pt-BR') ?? fallback
}

export function formatCurrencyCents(value: number | null | undefined) {
  if (typeof value !== 'number' || Number.isNaN(value)) return 'R$ 0,00'
  return (value / 100).toLocaleString('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  })
}

export function formatChaveAcesso(chave: string) {
  if (!chave) return ''
  const clean = chave.replace(/^NFS/i, '')
  return clean.length > 10 ? `...${clean.slice(-10)}` : clean
}

// formatChaveNFe prints a 44-character NF-e access key in the DANFE layout:
// 11 groups of 4 separated by spaces. Anything else is returned unchanged.
export function formatChaveNFe(chave: string | null | undefined) {
  if (!chave) return ''
  const clean = chave.replace(/\s/g, '')
  if (clean.length !== 44) return chave
  return (clean.match(/.{4}/g) ?? []).join(' ')
}

// formatNFeNumber prints the note number zero-padded to 9 digits and grouped
// by thousands, followed by the série: "000.012.345 / série 1".
export function formatNFeNumber(numero: string | null | undefined, serie?: string | null) {
  const digits = (numero ?? '').trim()
  let formatted = digits
  if (/^\d{1,9}$/.test(digits)) {
    formatted = digits.padStart(9, '0').replace(/^(\d{3})(\d{3})(\d{3})$/, '$1.$2.$3')
  }

  const serieValue = (serie ?? '').trim()
  if (!serieValue) return formatted
  const serieLabel = /^\d+$/.test(serieValue) ? String(Number(serieValue)) : serieValue
  return formatted ? `${formatted} / série ${serieLabel}` : `série ${serieLabel}`
}

// formatTime prints a local HH:MM time, or fallback for empty/invalid values.
export function formatTime(value: string | Date | null | undefined, fallback = '') {
  return parseDate(value)?.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }) ?? fallback
}
