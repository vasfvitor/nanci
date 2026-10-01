import type { CTeEvent, CTeMunicipio, CTeRow, CTeStatusResult } from '@/types/desktop'
import { formatDateTime } from '@/utils/formatters'
import { displayTable, type StateBadge } from '@/utils/sefazDisplay'

export const cteSituacao = displayTable({
  autorizada: { label: 'Autorizada', color: 'positive', abbr: 'A' },
  denegada: { label: 'Denegada', color: 'negative', abbr: 'D' },
  cancelada: { label: 'Cancelada', color: 'negative', abbr: 'C' },
})

// The roles are listed in priority order, as the backend picks the primary one.
export const ctePapel = displayTable({
  tomador: { label: 'Tomador', color: 'accent', abbr: 'TO' },
  destinatario: { label: 'Destinatário', color: 'secondary', abbr: 'DE' },
  remetente: { label: 'Remetente', color: 'info', abbr: 'RE' },
  expedidor: { label: 'Expedidor', color: 'info', abbr: 'EX' },
  recebedor: { label: 'Recebedor', color: 'info', abbr: 'RC' },
  emitente: { label: 'Emitente', color: 'primary', abbr: 'EM' },
  autorizado: { label: 'Autorizado', color: 'info', abbr: 'AU' },
  none: { label: 'Sem papel fiscal', color: 'grey', abbr: 'SP' },
})

export const cteModelo = displayTable(
  {
    '57': { label: 'CT-e', color: 'primary', abbr: 'CT' },
    '64': { label: 'GTV-e', color: 'accent', abbr: 'GV' },
    '67': { label: 'CT-e OS', color: 'secondary', abbr: 'OS' },
  },
  (modelo) => `Modelo ${modelo}`
)

export const cteTipoDocumento = displayTable({
  cte: { label: 'CT-e', color: 'primary', abbr: 'CT' },
  cte_os: { label: 'CT-e OS', color: 'secondary', abbr: 'OS' },
  gtve: { label: 'GTV-e', color: 'accent', abbr: 'GV' },
  cte_simplificado: { label: 'CT-e Simplificado', color: 'info', abbr: 'CS' },
})

// tpServ codes of the CT-e (0 to 4) and of the CT-e OS (6 to 8).
const cteTpServ = displayTable(
  {
    '0': { label: 'Normal', color: 'grey' },
    '1': { label: 'Subcontratação', color: 'grey' },
    '2': { label: 'Redespacho', color: 'grey' },
    '3': { label: 'Redespacho intermediário', color: 'grey' },
    '4': { label: 'Vinculado a multimodal', color: 'grey' },
    '6': { label: 'Transporte de pessoas', color: 'grey' },
    '7': { label: 'Transporte de valores', color: 'grey' },
    '8': { label: 'Excesso de bagagem', color: 'grey' },
  },
  (tpServ) => `Serviço ${tpServ}`
)

const cteModal = displayTable(
  {
    '01': { label: 'Rodoviário', color: 'grey' },
    '02': { label: 'Aéreo', color: 'grey' },
    '03': { label: 'Aquaviário', color: 'grey' },
    '04': { label: 'Ferroviário', color: 'grey' },
    '05': { label: 'Dutoviário', color: 'grey' },
    '06': { label: 'Multimodal', color: 'grey' },
  },
  (modal) => `Modal ${modal}`
)

// Keyed by CTeEvent.Type, the backend name of the tpEvento.
const cteEvent = displayTable({
  cancelamento: { label: 'Cancelamento', color: 'negative' },
  carta_correcao: { label: 'Carta de Correção', color: 'info' },
  epec: { label: 'EPEC', color: 'warning' },
  registro_multimodal: { label: 'Registro multimodal', color: 'info' },
  gtv: { label: 'Informações da GTV', color: 'info' },
  comprovante_entrega: { label: 'Comprovante de entrega', color: 'positive' },
  cancelamento_comprovante_entrega: { label: 'Cancelamento do comprovante de entrega', color: 'warning' },
  insucesso_entrega: { label: 'Insucesso na entrega', color: 'warning' },
  cancelamento_insucesso_entrega: { label: 'Cancelamento do insucesso na entrega', color: 'warning' },
  prestacao_desacordo: { label: 'Prestação de serviço em desacordo', color: 'negative' },
  cancelamento_desacordo: { label: 'Cancelamento da prestação em desacordo', color: 'warning' },
  mdfe_autorizado: { label: 'MDF-e autorizado', color: 'info' },
  mdfe_cancelado: { label: 'MDF-e cancelado', color: 'warning' },
  unknown: { label: 'Evento não reconhecido', color: 'grey' },
})

export const cteTpServLabel = cteTpServ.label
export const cteModalLabel = cteModal.label
export const cteEventColor = cteEvent.color

export const cteSituacaoFilterOptions = cteSituacao.options('Todas')
export const ctePapelFilterOptions = ctePapel.options('Todos')
export const cteModeloFilterOptions = cteModelo.options('Todos')

type CTeKind = Pick<CTeRow, 'Modelo' | 'TipoDocumento'>

// cteDocumentBadge names the kind of document: the tipo when known, so a
// CT-e Simplificado is not shown as a plain CT-e, else the modelo.
function cteDocumentBadge(row: CTeKind) {
  return row.TipoDocumento
    ? cteTipoDocumento.badge(row.TipoDocumento, 'Documento')
    : cteModelo.badge(row.Modelo, 'Documento')
}

// cteStateBadges are the state badges of a CT-e row, in the order of the
// table and the legend: documento, situação, papel principal, then the
// other papéis as secondary badges.
export function cteStateBadges(
  row: CTeKind & Pick<CTeRow, 'Situacao' | 'CompanyRole' | 'Papeis'>
): StateBadge[] {
  return [
    cteDocumentBadge(row),
    cteSituacao.badge(row.Situacao, 'Situação'),
    ctePapel.badge(row.CompanyRole, 'Papel'),
    ...cteOtherPapeis(row).map((papel) => ctePapel.badge(papel, 'Papel', true)),
  ]
}

// cteEventTitle names an event; one nanci does not recognize falls back to
// the description SEFAZ sent, then to its tpEvento code.
export function cteEventTitle(event: Pick<CTeEvent, 'Type' | 'TpEvento' | 'Description'>) {
  if (event.Type && event.Type !== 'unknown') return cteEvent.label(event.Type)
  if (event.Description) return event.Description
  return event.TpEvento ? `Evento ${event.TpEvento}` : cteEvent.label('unknown')
}

type CTeStatusCounts = Pick<
  CTeStatusResult,
  'TotalTomador' | 'TotalDestinatario' | 'TotalRemetente' | 'TotalOutros'
>

// cteDocumentCount counts the company's CT-e in every role.
export function cteDocumentCount(status: CTeStatusCounts | null) {
  return (
    (status?.TotalTomador ?? 0) +
    (status?.TotalDestinatario ?? 0) +
    (status?.TotalRemetente ?? 0) +
    (status?.TotalOutros ?? 0)
  )
}

// cteStatusLine sums up the last sync, the NSU cursor and the CT-e count.
export function cteStatusLine(status: CTeStatusResult) {
  return [
    `Última sincronização: ${formatDateTime(status.LastSyncAt, 'nunca')}`,
    `NSU ${status.LastNSU}/${status.MaxNSU ?? '—'}`,
    `CT-e: ${cteDocumentCount(status)}`,
  ].join(' · ')
}

// cteMunicipioLabel names a município as "Nome/UF", falling back to its IBGE
// code.
export function cteMunicipioLabel(municipio: CTeMunicipio) {
  if (!municipio.Nome) return municipio.Codigo || '—'
  return municipio.UF ? `${municipio.Nome}/${municipio.UF}` : municipio.Nome
}

// ctePercurso is where the transport starts and ends.
export function ctePercurso(row: Pick<CTeRow, 'MunIni' | 'MunFim'>) {
  return `${cteMunicipioLabel(row.MunIni)} → ${cteMunicipioLabel(row.MunFim)}`
}

export type CTeParticipante = { label: string; cnpj: string; name: string }

// cteParticipantes lists the parties the CT-e names besides the emitente,
// in the XML order, leaving out the absent ones.
export function cteParticipantes(row: CTeRow): CTeParticipante[] {
  return [
    { label: 'Remetente', cnpj: row.RemetenteCNPJ, name: row.RemetenteName },
    { label: 'Destinatário', cnpj: row.DestinatarioCNPJ, name: row.DestinatarioName },
    { label: 'Expedidor', cnpj: row.ExpedidorCNPJ, name: row.ExpedidorName },
    { label: 'Recebedor', cnpj: row.RecebedorCNPJ, name: row.RecebedorName },
    { label: 'Tomador', cnpj: row.TomadorCNPJ, name: row.TomadorName },
  ].filter((party) => party.cnpj || party.name)
}

// cteOtherPapeis are the roles the company plays besides the primary one.
export function cteOtherPapeis(row: Pick<CTeRow, 'CompanyRole' | 'Papeis'>) {
  return row.Papeis.filter((papel) => papel !== row.CompanyRole)
}
