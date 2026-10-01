import type { NFeRow, NFeStatusResult } from '@/types/desktop'
import { formatDateTime } from '@/utils/formatters'
import { displayTable, type StateBadge } from '@/utils/sefazDisplay'

export type DeadlineKind = 'ciencia' | 'conclusiva'

// Days left before a deadline at which a chip turns warning, then urgent.
// The ciência window is only 10 days, so it gets tighter thresholds.
export const DEADLINE_THRESHOLDS: Record<DeadlineKind, { warning: number; urgent: number }> = {
  ciencia: { warning: 10, urgent: 3 },
  conclusiva: { warning: 30, urgent: 10 },
}

export const nfeSituacao = displayTable({
  autorizada: { label: 'Autorizada', color: 'positive', abbr: 'A' },
  denegada: { label: 'Denegada', color: 'negative', abbr: 'D' },
  cancelada: { label: 'Cancelada', color: 'negative', abbr: 'C' },
})

export const nfeCompleteness = displayTable({
  resumo: { label: 'Resumo', color: 'warning', abbr: 'R' },
  completa: { label: 'Completa', color: 'positive', abbr: 'X' },
})

export const nfeManifestacao = displayTable({
  nenhuma: { label: 'Sem manifestação', color: 'grey', abbr: 'SM' },
  ciencia: { label: 'Ciência', color: 'info', abbr: 'CI' },
  confirmada: { label: 'Confirmada', color: 'positive', abbr: 'CO' },
  desconhecida: { label: 'Desconhecida', color: 'negative', abbr: 'DE' },
  nao_realizada: { label: 'Operação não realizada', color: 'warning', abbr: 'NR' },
})

export const nfeRole = displayTable({
  destinatario: { label: 'Destinatário', color: 'secondary', abbr: 'D' },
  emitente: { label: 'Emitente', color: 'primary', abbr: 'E' },
  transportador: { label: 'Transportador', color: 'accent', abbr: 'T' },
  autorizado: { label: 'Autorizado', color: 'info', abbr: 'A' },
  none: { label: 'Sem papel fiscal', color: 'grey', abbr: 'SP' },
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

export const situacaoLabel = nfeSituacao.label
export const nfeEventLabel = nfeEvent.label
export const nfeEventColor = nfeEvent.color
export const outcomeLabel = outcome.label
export const outcomeColor = outcome.color

export const situacaoFilterOptions = nfeSituacao.options('Todas')
export const completenessFilterOptions = nfeCompleteness.options('Todas')
export const manifestacaoFilterOptions = nfeManifestacao.options('Todas')
export const nfeRoleFilterOptions = nfeRole.options('Todos')

// nfeStateBadges are the state badges of an NF-e row, in the order of the
// table and the legend: situação, completude, manifestação, papel.
export function nfeStateBadges(
  row: Pick<NFeRow, 'Situacao' | 'Completeness' | 'Manifestacao' | 'CompanyRole'>
): StateBadge[] {
  return [
    nfeSituacao.badge(row.Situacao, 'Situação'),
    nfeCompleteness.badge(row.Completeness, 'Completude'),
    nfeManifestacao.badge(row.Manifestacao, 'Manifestação'),
    nfeRole.badge(row.CompanyRole, 'Papel'),
  ]
}

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
