import type { CTeRow } from '@/types/desktop'

// Limits of the observação of the prestação em desacordo
// (evPrestDesacordo_v4.00.xsd).
export const OBSERVACAO_MIN_LENGTH = 15
export const OBSERVACAO_MAX_LENGTH = 255

// validateObservacao applies the schema rule the backend checks again
// (sefaz.CheckXObsDesacordo): 15 to 255 characters within Latin-1, without a
// leading or trailing space. It returns the problem, or null.
export function validateObservacao(text: string): string | null {
  const length = [...text].length
  if (length < OBSERVACAO_MIN_LENGTH || length > OBSERVACAO_MAX_LENGTH) {
    return `Use de ${OBSERVACAO_MIN_LENGTH} a ${OBSERVACAO_MAX_LENGTH} caracteres.`
  }
  if (text.trim() !== text) {
    return 'Tire os espaços do começo e do fim.'
  }
  for (const char of text) {
    const code = char.codePointAt(0) ?? 0
    if (code < 0x20 || code > 0xff || (code >= 0x7f && code < 0xa0)) {
      return `A SEFAZ não aceita o caractere "${char}". Use aspas retas e evite emoji.`
    }
  }
  return null
}

// desacordoMenuBlock says why the row menu cannot offer the prestação em
// desacordo, or null. The backend checks again, with the exact tomador CNPJ.
export function desacordoMenuBlock(
  row: Pick<CTeRow, 'Modelo' | 'CompanyRole' | 'Papeis' | 'Situacao'>
): string | null {
  if (row.Modelo !== '57' && row.Modelo !== '67') {
    return 'Só para CT-e e CT-e OS'
  }
  if (row.CompanyRole !== 'tomador' && !row.Papeis.includes('tomador')) {
    return 'Só o tomador pode enviar'
  }
  if (row.Situacao !== 'autorizada') {
    return `CT-e ${row.Situacao || 'sem situação'}`
  }
  return null
}

// desacordoDeadlineText describes the 45 days from the authorization.
export function desacordoDeadlineText(daysLeft: number | null): string {
  if (daysLeft === null) return 'sem data de autorização'
  if (daysLeft < 0) return `vencido há ${-daysLeft} dia(s)`
  if (daysLeft === 0) return 'vence hoje'
  return `faltam ${daysLeft} dia(s)`
}
