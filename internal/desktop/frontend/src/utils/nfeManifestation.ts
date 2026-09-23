import type { NFeConclusiveTipo, NFeRow } from '@/types/desktop'

// Rules that enable or disable manifestação actions in the UI. The backend
// enforces them again, and SEFAZ has the final word on deadlines.

export const JUSTIFICATIVA_MIN_LENGTH = 15
export const JUSTIFICATIVA_MAX_LENGTH = 255

// CONCLUSIVE_TIPOS are the conclusive manifestação events, in the order the
// UI offers them.
export const CONCLUSIVE_TIPOS: readonly NFeConclusiveTipo[] = ['210200', '210220', '210240']

type ManifestableRow = Pick<NFeRow, 'CompanyRole' | 'Situacao' | 'Manifestacao'>

function noteBlockReason(row: ManifestableRow): string | null {
  if (row.CompanyRole !== 'destinatario') return 'Somente o destinatário pode manifestar'
  if (row.Situacao === 'cancelada') return 'Nota cancelada'
  if (row.Situacao === 'denegada') return 'Nota denegada'
  if (row.Situacao !== 'autorizada') return 'Situação da nota desconhecida'
  if (row.Manifestacao === '') return 'Manifestação atual desconhecida'
  return null
}

export function cienciaBlockReason(row: ManifestableRow): string | null {
  const reason = noteBlockReason(row)
  if (reason) return reason
  if (row.Manifestacao !== 'nenhuma') return 'Já possui manifestação'
  return null
}

// conclusiveBlockReason applies to every conclusive tipo alike: only one
// conclusive manifestação can be registered per note.
export function conclusiveBlockReason(row: ManifestableRow): string | null {
  const reason = noteBlockReason(row)
  if (reason) return reason
  if (row.Manifestacao !== 'nenhuma' && row.Manifestacao !== 'ciencia') {
    return 'Já possui manifestação conclusiva'
  }
  return null
}

export function validateJustificativa(text: string | null | undefined): string | null {
  const length = (text ?? '').trim().length
  if (length < JUSTIFICATIVA_MIN_LENGTH) {
    return `A justificativa deve ter pelo menos ${JUSTIFICATIVA_MIN_LENGTH} caracteres`
  }
  if (length > JUSTIFICATIVA_MAX_LENGTH) {
    return `A justificativa deve ter no máximo ${JUSTIFICATIVA_MAX_LENGTH} caracteres`
  }
  return null
}
