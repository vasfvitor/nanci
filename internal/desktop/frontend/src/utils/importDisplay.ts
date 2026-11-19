import { formatCount } from '@/utils/formatters'
import type { ImportXMLResult } from '@/types/desktop'
import { displayTable } from '@/utils/sefazDisplay'

const importOutcome = displayTable({
  imported: { label: 'Importado', color: 'positive' },
  updated: { label: 'Atualizado', color: 'info' },
  rejected: { label: 'Rejeitado', color: 'negative' },
})

const importKind = displayTable({
  completa: { label: 'Completa', color: 'positive' },
  resumo: { label: 'Resumo', color: 'warning' },
  evento: { label: 'Evento', color: 'info' },
})

export const importOutcomeLabel = importOutcome.label
export const importOutcomeColor = importOutcome.color

// importKindLabel names what an imported file held; a file rejected before
// it was read has no kind and shows "—".
export function importKindLabel(kind: string) {
  return kind ? importKind.label(kind) : '—'
}

// fileName is the last part of a Windows or POSIX path.
export function fileName(path: string) {
  return path.split(/[\\/]/).pop() ?? path
}

// importSummary counts the files of an import by outcome:
// "2 importados, 1 atualizado, 0 rejeitados".
export function importSummary(result: Pick<ImportXMLResult, 'Imported' | 'Updated' | 'Rejected'>) {
  return [
    formatCount(result.Imported, 'importado', 'importados'),
    formatCount(result.Updated, 'atualizado', 'atualizados'),
    formatCount(result.Rejected, 'rejeitado', 'rejeitados'),
  ].join(', ')
}
