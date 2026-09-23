export type DeadlineKind = 'ciencia' | 'conclusiva'

// Days left before a deadline at which a chip turns warning, then urgent.
// The ciência window is only 10 days, so it gets tighter thresholds.
export const DEADLINE_THRESHOLDS: Record<DeadlineKind, { warning: number; urgent: number }> = {
  ciencia: { warning: 10, urgent: 3 },
  conclusiva: { warning: 30, urgent: 10 },
}

type Display = { label: string; color: string }

export type FilterOption = { label: string; value: string }

// displayTable looks values up in table. An unknown value is shown as
// unknownLabel(value) in grey, and an empty one as "Desconhecido".
function displayTable(
  table: Record<string, Display>,
  unknownLabel: (value: string) => string = (value) => value
) {
  const find = (value: string) => (Object.hasOwn(table, value) ? table[value] : undefined)
  return {
    label: (value: string) => find(value)?.label ?? (value ? unknownLabel(value) : 'Desconhecido'),
    color: (value: string) => find(value)?.color ?? 'grey',
    // options lists every known value for a filter, after an empty "all" entry.
    options: (allLabel: string): FilterOption[] => [
      { label: allLabel, value: '' },
      ...Object.entries(table).map(([value, display]) => ({ label: display.label, value })),
    ],
  }
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
    '210240': { label: 'Operação não Realizada', color: 'warning' },
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

// deadlineColor colors a days-left chip; days comes from daysUntil.
export function deadlineColor(days: number | null, kind: DeadlineKind) {
  if (days === null) return 'grey'
  const thresholds = DEADLINE_THRESHOLDS[kind]
  if (days <= thresholds.urgent) return 'negative'
  if (days <= thresholds.warning) return 'warning'
  return 'grey'
}

// badgeColor is the fill of a badge of color. In light mode the info blue
// is too light for white text, so it uses a darker shade.
export function badgeColor(color: string, dark: boolean) {
  if (!dark && color === 'info') return 'light-blue-9'
  return color
}

// badgeTextColor picks a readable text color for a filled badge of color.
// In dark mode every palette color is light, so text is dark; in light mode
// only warning and grey are too light for white text.
export function badgeTextColor(color: string, dark: boolean) {
  if (dark || color === 'warning' || color === 'grey') return 'dark'
  return 'white'
}

// deadlineLabel describes the days left from daysUntil.
export function deadlineLabel(days: number | null) {
  if (days === null) return 'Sem prazo'
  if (days < 0) return `Vencido há ${-days} d`
  if (days === 0) return 'Vence hoje'
  return `${days} d restantes`
}

export const TACIT_CONFIRMATION_LABEL = 'Confirmada tacitamente'

// conclusiveDeadlineLabel is deadlineLabel for the conclusive manifestação
// deadline: once it has passed, the operation is deemed confirmed by law.
export function conclusiveDeadlineLabel(days: number | null) {
  if (days !== null && days < 0) return TACIT_CONFIRMATION_LABEL
  return deadlineLabel(days)
}

// ambienteColor highlights production (tpAmb 1), where events are fiscal acts.
export function ambienteColor(tpAmb: string) {
  if (tpAmb === '1') return 'negative'
  if (tpAmb === '2') return 'warning'
  return 'grey'
}

export type NFeBlockInfo = {
  BlockedReason: string
  RequestsLastHour: number
  RequestBudget: number
}

// blockedMessage explains why NF-e sync is paused; time is the HH:MM the
// block ends.
export function blockedMessage(info: NFeBlockInfo, time: string) {
  switch (info.BlockedReason) {
    case 'caught_up':
      return `Sem novos documentos; próxima consulta a partir de ${time}`
    case 'consumo_indevido':
      return `Consultas bloqueadas pela SEFAZ até ${time} (cStat 656)`
    case 'rate_budget':
      return `Limite de consultas por hora atingido (${info.RequestsLastHour}/${info.RequestBudget}); aguarde até ${time}`
    default:
      return `Próxima consulta permitida a partir de ${time}`
  }
}
