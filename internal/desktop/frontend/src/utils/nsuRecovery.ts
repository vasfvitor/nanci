import type { RecoverNSUResult } from '@/types/desktop'
import { formatCount } from '@/utils/formatters'

function nsus(count: number) {
  return formatCount(count, 'NSU', 'NSUs')
}

// missingNSUsText is the notice above the table.
export function missingNSUsText(missing: number) {
  return `${nsus(missing)} faltando`
}

// recoveryConfirmText explains, before the run, what it spends.
export function recoveryConfirmText(missing: number, allowance: number, budget: number) {
  return (
    `Faltam ${nsus(missing)}. A recuperação pede cada um à SEFAZ (consNSU) ` +
    `e usa até ${allowance} das ${budget} consultas desta hora; as outras ficam para a sincronização. ` +
    'O que faltar continua na próxima vez.'
  )
}

// recoverySummary is the notice after the run.
export function recoverySummary(result: RecoverNSUResult) {
  if (result.Asked === 0 && result.Expired === 0 && result.Missing === 0) {
    return 'Nenhum NSU faltando.'
  }
  const parts = [
    formatCount(result.Asked, 'NSU consultado', 'NSUs consultados'),
    `${result.Recovered} com documento`,
    `${result.Empty} sem documento`,
  ]
  if (result.Unreadable > 0) parts.push(`${result.Unreadable} ilegíveis (veja o log)`)
  if (result.Expired > 0) parts.push(`${result.Expired} fora da janela de 90 dias`)
  let text = `${parts.join(', ')}.`
  if (result.Missing > 0) {
    text += ` Ainda ${result.Missing === 1 ? 'falta' : 'faltam'} ${nsus(result.Missing)}.`
  }
  if (result.StopReason === 'consumo_indevido') {
    text += ' A SEFAZ bloqueou o CNPJ por consumo indevido (656); tente depois do horário indicado.'
  } else if (result.StopReason === 'rate_budget') {
    text += ' As consultas desta hora acabaram.'
  } else if (result.StopReason === 'canceled') {
    text += ' Recuperação cancelada; o que falta continua na próxima vez.'
  }
  return text
}
