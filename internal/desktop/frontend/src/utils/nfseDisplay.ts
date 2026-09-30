import type { CompanySummary, DocumentRow } from '@/types/desktop'
import { formatDateTime } from '@/utils/formatters'
import { ambienteColor, displayTable, type StateBadge } from '@/utils/sefazDisplay'

const nfseStatusEntries = {
  normal: { label: 'Normal', color: 'positive', abbr: 'N' },
  cancelada: { label: 'Cancelada', color: 'negative', abbr: 'C' },
  substituida: { label: 'Substituída', color: 'negative', abbr: 'S' },
}

// Why the ADN delivered the NFS-e to the company.
const nfseVisibilityEntries = {
  exact_prestador: { label: 'Prestador exato', color: 'positive', abbr: 'PE' },
  exact_tomador: { label: 'Tomador exato', color: 'positive', abbr: 'TE' },
  exact_intermediario: { label: 'Intermediário exato', color: 'positive', abbr: 'IE' },
  same_root_only: { label: 'Mesmo raiz apenas', color: 'warning', abbr: 'MR' },
  unknown: { label: 'Desconhecida', color: 'grey', abbr: '?' },
}

const nfseRoleEntries = {
  prestada: { label: 'Prestada', color: 'primary', abbr: 'P' },
  tomada: { label: 'Tomada', color: 'secondary', abbr: 'T' },
  intermediario: { label: 'Intermediário', color: 'accent', abbr: 'I' },
  none: { label: 'Sem papel fiscal', color: 'grey', abbr: 'SP' },
}

export const nfseStatus = displayTable(nfseStatusEntries)
export const nfseVisibility = displayTable(nfseVisibilityEntries)
export const nfseRole = displayTable(nfseRoleEntries)

// Keyed by DocumentEvent.Type.
export const nfseEvent = displayTable({
  cancelamento: { label: 'Cancelamento', color: 'negative' },
  substituicao: { label: 'Substituição', color: 'warning' },
  unknown: { label: 'Evento não reconhecido', color: 'grey' },
})

// nfseStateBadges are the state badges of an NFS-e row, in the order of the
// table and the legend: status, visibilidade, papel.
export function nfseStateBadges(
  row: Pick<DocumentRow, 'Status' | 'VisibilityReason' | 'CompanyRole'>
): StateBadge[] {
  return [
    nfseStatus.badge(row.Status, 'Status'),
    nfseVisibility.badge(row.VisibilityReason, 'Visibilidade'),
    nfseRole.badge(row.CompanyRole, 'Papel'),
  ]
}

// nfseAmbiente names the ADN environment of a company, colored like the
// SEFAZ tpAmb it matches: production is where documents are fiscal acts.
export function nfseAmbiente(environment: string) {
  if (environment === 'producao') return { label: 'Produção', color: ambienteColor('1') }
  if (environment === 'producao_restrita') {
    return { label: 'Produção restrita', color: ambienteColor('2') }
  }
  return { label: 'Ambiente desconhecido', color: 'grey' }
}

// nfseStatusLine sums up the company's last NFS-e sync and the last NSU that
// brought a document.
export function nfseStatusLine(company: Pick<CompanySummary, 'LastSyncAt' | 'LastFoundNSU'>) {
  return [
    `Última sincronização: ${formatDateTime(company.LastSyncAt, 'nunca')}`,
    `NSU ${company.LastFoundNSU ?? '—'}`,
  ].join(' · ')
}
