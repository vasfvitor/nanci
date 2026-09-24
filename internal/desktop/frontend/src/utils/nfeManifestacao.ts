import type { NFeConclusiveTipo, NFeEventOutcome, NFeEventResult, NFeRow } from '@/types/desktop'

// The backend decides which manifestação a note can receive and says why
// not in the row; SEFAZ has the final word on deadlines. The frontend only
// validates the justificativa as it is typed.

export const JUSTIFICATIVA_MIN_LENGTH = 15
export const JUSTIFICATIVA_MAX_LENGTH = 255

// CONCLUSIVE_TIPOS are the conclusive manifestação events, in the order the
// UI offers them.
export const CONCLUSIVE_TIPOS: readonly NFeConclusiveTipo[] = ['210200', '210220', '210240']

// displayReason starts a backend reason with a capital letter, or returns
// null when there is none.
function displayReason(reason: string): string | null {
  return reason ? reason.charAt(0).toUpperCase() + reason.slice(1) : null
}

export function cienciaBlockReason(row: Pick<NFeRow, 'CienciaBlockReason'>): string | null {
  return displayReason(row.CienciaBlockReason)
}

// conclusiveBlockReason applies to every conclusive tipo alike: only one
// conclusive manifestação can be registered per note.
export function conclusiveBlockReason(row: Pick<NFeRow, 'ConclusiveBlockReason'>): string | null {
  return displayReason(row.ConclusiveBlockReason)
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

// countOutcomes counts ciência results by outcome. A result with an unknown
// status counts as not sent.
export function countOutcomes(results: readonly NFeEventResult[]): Record<NFeEventOutcome, number> {
  const counts = { registrada: 0, ja_registrada: 0, rejeitada: 0, nao_enviada: 0 }
  for (const result of results) counts[result.Status || 'nao_enviada']++
  return counts
}
