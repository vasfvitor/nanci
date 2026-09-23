// Days left before a manifestação deadline at which the UI starts warning.
export const DEADLINE_WARNING_DAYS = 30
export const DEADLINE_URGENT_DAYS = 10

export const situacaoLabels: Record<string, string> = {
  autorizada: 'Autorizada',
  denegada: 'Denegada',
  cancelada: 'Cancelada',
}

export const situacaoColors: Record<string, string> = {
  autorizada: 'positive',
  denegada: 'negative',
  cancelada: 'negative',
}

export const completenessLabels: Record<string, string> = {
  resumo: 'Resumo',
  completa: 'Completa',
}

export const completenessColors: Record<string, string> = {
  resumo: 'warning',
  completa: 'positive',
}

export const manifestacaoLabels: Record<string, string> = {
  nenhuma: 'Sem manifestação',
  ciencia: 'Ciência',
  confirmada: 'Confirmada',
  desconhecida: 'Desconhecida',
  nao_realizada: 'Operação não realizada',
}

export const manifestacaoColors: Record<string, string> = {
  nenhuma: 'grey',
  ciencia: 'info',
  confirmada: 'positive',
  desconhecida: 'negative',
  nao_realizada: 'warning',
}

export const nfeRoleLabels: Record<string, string> = {
  destinatario: 'Destinatário',
  emitente: 'Emitente',
  transportador: 'Transportador',
  autorizado: 'Autorizado',
  none: 'Sem papel fiscal',
}

export const nfeRoleColors: Record<string, string> = {
  destinatario: 'secondary',
  emitente: 'primary',
  transportador: 'accent',
  autorizado: 'info',
  none: 'grey',
}

export const nfeEventLabels: Record<string, string> = {
  '110111': 'Cancelamento',
  '110110': 'Carta de Correção',
  '210210': 'Ciência',
  '210200': 'Confirmação',
  '210220': 'Desconhecimento',
  '210240': 'Operação não Realizada',
}

export const nfeEventColors: Record<string, string> = {
  '110111': 'negative',
  '110110': 'info',
  '210210': 'info',
  '210200': 'positive',
  '210220': 'negative',
  '210240': 'warning',
}

export const outcomeLabels: Record<string, string> = {
  registrada: 'Registrada',
  ja_registrada: 'Já registrada',
  rejeitada: 'Rejeitada',
  nao_enviada: 'Não enviada',
}

export const outcomeColors: Record<string, string> = {
  registrada: 'positive',
  ja_registrada: 'info',
  rejeitada: 'negative',
  nao_enviada: 'warning',
}

export function situacaoLabel(situacao: string) {
  return situacaoLabels[situacao] || situacao || 'Desconhecido'
}

export function situacaoColor(situacao: string) {
  return situacaoColors[situacao] || 'grey'
}

export function completenessLabel(completeness: string) {
  return completenessLabels[completeness] || completeness || 'Desconhecido'
}

export function completenessColor(completeness: string) {
  return completenessColors[completeness] || 'grey'
}

export function manifestacaoLabel(manifestacao: string) {
  return manifestacaoLabels[manifestacao] || manifestacao || 'Desconhecido'
}

export function manifestacaoColor(manifestacao: string) {
  return manifestacaoColors[manifestacao] || 'grey'
}

export function nfeRoleLabel(role: string) {
  return nfeRoleLabels[role] || role || 'Desconhecido'
}

export function nfeRoleColor(role: string) {
  return nfeRoleColors[role] || 'grey'
}

export function nfeEventLabel(tpEvento: string) {
  if (!tpEvento) return 'Desconhecido'
  return nfeEventLabels[tpEvento] || `Evento ${tpEvento}`
}

export function nfeEventColor(tpEvento: string) {
  return nfeEventColors[tpEvento] || 'grey'
}

export function outcomeLabel(outcome: string) {
  return outcomeLabels[outcome] || outcome || 'Desconhecido'
}

export function outcomeColor(outcome: string) {
  return outcomeColors[outcome] || 'grey'
}

// deadlineColor colors a days-left chip; days comes from daysUntil.
export function deadlineColor(days: number | null) {
  if (days === null) return 'grey'
  if (days <= DEADLINE_URGENT_DAYS) return 'negative'
  if (days <= DEADLINE_WARNING_DAYS) return 'warning'
  return 'grey'
}

// deadlineLabel describes the days left from daysUntil.
export function deadlineLabel(days: number | null) {
  if (days === null) return 'Sem prazo'
  if (days < 0) return `Vencido há ${-days} d`
  if (days === 0) return 'Vence hoje'
  return `${days} d restantes`
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
