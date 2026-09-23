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

export function formatDate(value: string | Date | null | undefined) {
  if (!value) return ''
  const parsed = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(parsed.getTime())) return ''
  return parsed.toLocaleDateString('pt-BR')
}

export function formatDateTime(value: string | Date | null | undefined, fallback = '-') {
  if (!value) return fallback
  const parsed = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(parsed.getTime())) return fallback
  return parsed.toLocaleString('pt-BR')
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

// daysUntil counts whole local calendar days from now until value: 0 for
// today, negative when the date has passed, null when value is empty or invalid.
export function daysUntil(value: string | Date | null | undefined, now: Date = new Date()) {
  if (!value) return null
  const target = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(target.getTime()) || Number.isNaN(now.getTime())) return null
  const targetDay = Date.UTC(target.getFullYear(), target.getMonth(), target.getDate())
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())
  return Math.round((targetDay - today) / 86_400_000)
}
