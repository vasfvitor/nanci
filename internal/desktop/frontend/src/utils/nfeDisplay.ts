import type { NFeRow, NFeStatusResult } from '@/types/desktop'
import { formatDateTime } from '@/utils/formatters'
import { displayTable } from '@/utils/sefazDisplay'

export type DeadlineKind = 'ciencia' | 'conclusiva'

// Days left before a deadline at which a chip turns warning, then urgent.
// The ciência window is only 10 days, so it gets tighter thresholds.
export const DEADLINE_THRESHOLDS: Record<DeadlineKind, { warning: number; urgent: number }> = {
  ciencia: { warning: 10, urgent: 3 },
  conclusiva: { warning: 30, urgent: 10 },
}

const situacao = displayTable({
  autorizada: { label: 'Autorizada', color: 'positive' },
  denegada: { label: 'Denegada', color: 'negative' },
  cancelada: { label: 'Cancelada', color: 'negative' },
})

const completeness = displayTable({
  resumo: { label: 'Resumo', color: 'warning' },
  completa: { label: 'Completa', color: 'positive' },
})

const manifestacao = displayTable({
  nenhuma: { label: 'Sem manifestação', color: 'grey' },
  ciencia: { label: 'Ciência', color: 'info' },
  confirmada: { label: 'Confirmada', color: 'positive' },
  desconhecida: { label: 'Desconhecida', color: 'negative' },
  nao_realizada: { label: 'Operação não realizada', color: 'warning' },
})

const nfeRole = displayTable({
  destinatario: { label: 'Destinatário', color: 'secondary' },
  emitente: { label: 'Emitente', color: 'primary' },
  transportador: { label: 'Transportador', color: 'accent' },
  autorizado: { label: 'Autorizado', color: 'info' },
  none: { label: 'Sem papel fiscal', color: 'grey' },
})

const nfeEvent = displayTable(
  {
    '110111': { label: 'Cancelamento', color: 'negative' },
    '110110': { label: 'Carta de Correção', color: 'info' },
    '210210': { label: 'Ciência', color: 'info' },
    '210200': { label: 'Confirmação', color: 'positive' },
    '210220': { label: 'Desconhecimento', color: 'negative' },
    '210240': { label: 'Operação não realizada', color: 'warning' },
  },
  (tpEvento) => `Evento ${tpEvento}`
)

const outcome = displayTable({
  registrada: { label: 'Registrada', color: 'positive' },
  ja_registrada: { label: 'Já registrada', color: 'info' },
  rejeitada: { label: 'Rejeitada', color: 'negative' },
  nao_enviada: { label: 'Não enviada', color: 'warning' },
})

export const situacaoLabel = situacao.label
export const situacaoColor = situacao.color
export const completenessLabel = completeness.label
export const completenessColor = completeness.color
export const manifestacaoLabel = manifestacao.label
export const manifestacaoColor = manifestacao.color
export const nfeRoleLabel = nfeRole.label
export const nfeRoleColor = nfeRole.color
export const nfeEventLabel = nfeEvent.label
export const nfeEventColor = nfeEvent.color
export const outcomeLabel = outcome.label
export const outcomeColor = outcome.color

export const situacaoFilterOptions = situacao.options('Todas')
export const completenessFilterOptions = completeness.options('Todas')
export const manifestacaoFilterOptions = manifestacao.options('Todas')
export const nfeRoleFilterOptions = nfeRole.options('Todos')

// deadlineColor colors a days-left chip; days is a row's DaysLeft or
// CienciaDaysLeft.
export function deadlineColor(days: number | null, kind: DeadlineKind) {
  if (days === null) return 'grey'
  const thresholds = DEADLINE_THRESHOLDS[kind]
  if (days <= thresholds.urgent) return 'negative'
  if (days <= thresholds.warning) return 'warning'
  return 'grey'
}

// deadlineLabel describes the days left before a deadline, as the backend
// counts them.
export function deadlineLabel(days: number | null) {
  if (days === null) return 'Sem prazo'
  if (days < 0) return `Vencido há ${-days} d`
  if (days === 0) return 'Vence hoje'
  return `${days} d restantes`
}

// showsConclusiveDeadline reports whether the grid shows the conclusive
// deadline chip: only a note with ciência still waits for a conclusive
// manifestação.
export function showsConclusiveDeadline(row: Pick<NFeRow, 'Manifestacao' | 'ConclusiveDue'>) {
  return row.Manifestacao === 'ciencia' && Boolean(row.ConclusiveDue)
}

export const TACIT_CONFIRMATION_LABEL = 'Confirmada tacitamente'

// conclusiveDeadlineLabel is deadlineLabel for the conclusive manifestação
// deadline: once it has passed, the operation is deemed confirmed by law.
export function conclusiveDeadlineLabel(days: number | null) {
  if (days !== null && days < 0) return TACIT_CONFIRMATION_LABEL
  return deadlineLabel(days)
}

type NFeStatusCounts = Pick<
  NFeStatusResult,
  'PendingCiencia' | 'PendingConclusiva' | 'TotalDestinatario' | 'TotalEmitente' | 'TotalOutros'
>

// nfePendingCount counts the notes still waiting for a manifestação.
export function nfePendingCount(status: NFeStatusCounts | null) {
  return (status?.PendingCiencia ?? 0) + (status?.PendingConclusiva ?? 0)
}

// nfeNoteCount counts the company's notes in every role.
export function nfeNoteCount(status: NFeStatusCounts | null) {
  return (status?.TotalDestinatario ?? 0) + (status?.TotalEmitente ?? 0) + (status?.TotalOutros ?? 0)
}

// nfeStatusLine sums up the last sync, the NSU cursor and the pendências.
export function nfeStatusLine(status: NFeStatusResult) {
  return [
    `Última sincronização: ${formatDateTime(status.LastSyncAt, 'nunca')}`,
    `NSU ${status.LastNSU}/${status.MaxNSU ?? '—'}`,
    `Pendências: ${nfePendingCount(status)}`,
  ].join(' · ')
}
