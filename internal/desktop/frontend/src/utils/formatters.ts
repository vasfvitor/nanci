export function formatCpfCnpj(value: string | null | undefined) {
  if (!value) return ''
  const digits = value.replace(/\D/g, '')
  if (digits.length === 14) {
    return digits.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, '$1.$2.$3/$4-$5')
  }
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
