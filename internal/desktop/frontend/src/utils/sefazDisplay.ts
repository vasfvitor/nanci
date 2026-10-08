// Display helpers shared by the document screens (NFS-e, NF-e and CT-e):
// display tables for the state values, badge colors and the SEFAZ ambiente.

// Display is how a state value is shown: its full label, its color, the
// short abbreviation the table badges carry and the description its legend
// entry gives.
type Display = { label: string; color: string; abbr?: string; description?: string }

export type LegendItem = {
  // Text inside the badge, as the table shows it.
  badge: string
  color: string
  // Full name when the badge is an abbreviation.
  name?: string
  description: string
  // Rendered as an outlined chip instead of a filled badge.
  outline?: boolean
}

export type LegendSection = {
  title: string
  // Optional line under the title, for notes that apply to the whole group.
  note?: string
  items: LegendItem[]
}

export type FilterOption = { label: string; value: string }

// StateBadge is one abbreviated badge of a document row. kind names its group
// ("Situação", "Papel"...), for the "Kind: Label" tooltip; key is unique
// within a row; secondary badges go on a second, smaller line.
export type StateBadge = {
  key: string
  abbr: string
  label: string
  color: string
  kind: string
  secondary?: boolean
}

export type DisplayTable = ReturnType<typeof displayTable>

// displayTable looks values up in table. An unknown value is shown as
// unknownLabel(value) in grey with the abbreviation "?", and an empty one as
// "Desconhecido" with "—".
export function displayTable(
  table: Record<string, Display>,
  unknownLabel: (value: string) => string = (value) => value
) {
  const find = (value: string) => (Object.hasOwn(table, value) ? table[value] : undefined)
  const label = (value: string) => find(value)?.label ?? (value ? unknownLabel(value) : 'Desconhecido')
  const color = (value: string) => find(value)?.color ?? 'grey'
  const abbr = (value: string) => (value ? (find(value)?.abbr ?? '?') : '—')
  return {
    label,
    color,
    abbr,
    // badge is the StateBadge of value in the group kind.
    badge: (value: string, kind: string, secondary = false): StateBadge => ({
      key: `${kind}:${value}`,
      abbr: abbr(value),
      label: label(value),
      color: color(value),
      kind,
      ...(secondary ? { secondary: true } : {}),
    }),
    // options lists every known value for a filter, after an empty "all" entry.
    options: (allLabel: string): FilterOption[] => [
      { label: allLabel, value: '' },
      ...Object.entries(table).map(([value, display]) => ({ label: display.label, value })),
    ],
    // values lists every known value, in table order.
    values: () => Object.keys(table),
    // legend lists the legend entries of the values that have a description,
    // in table order.
    legend: (): LegendItem[] =>
      Object.entries(table).flatMap(([value, { label, color, description }]) =>
        description ? [{ badge: abbr(value), color, name: label, description }] : []
      ),
  }
}

// VIEWED_BADGE is the badge of a document not viewed yet, in every table and
// in the legends.
export const VIEWED_BADGE = { label: 'Novo', color: 'warning' } as const

// badgeColor is the fill of a badge of color. In light mode the info blue
// is too light for white text, so it uses a darker shade.
function badgeColor(color: string, dark: boolean) {
  if (!dark && color === 'info') return 'light-blue-9'
  return color
}

// badgeTextColor picks a readable text color for a filled badge of color.
// In dark mode every palette color is light, so text is dark; in light mode
// only warning and grey are too light for white text.
function badgeTextColor(color: string, dark: boolean) {
  if (dark || color === 'warning' || color === 'grey') return 'dark'
  return 'white'
}

// badgeProps binds the fill and text color of a q-badge of color.
export function badgeProps(color: string, dark: boolean) {
  return { color: badgeColor(color, dark), textColor: badgeTextColor(color, dark) }
}

// ambienteLabel names the SEFAZ environment of tpAmb.
export function ambienteLabel(tpAmb: string) {
  if (tpAmb === '1') return 'Produção'
  if (tpAmb === '2') return 'Homologação'
  return 'Ambiente desconhecido'
}

// ambienteColor highlights production (tpAmb 1), where events are fiscal acts.
export function ambienteColor(tpAmb: string) {
  if (tpAmb === '1') return 'negative'
  if (tpAmb === '2') return 'warning'
  return 'grey'
}

// sefazAmbiente is the ambiente badge of a page header for tpAmb.
export function sefazAmbiente(tpAmb: string) {
  return { label: ambienteLabel(tpAmb), color: ambienteColor(tpAmb) }
}

export type SefazBlockInfo = {
  BlockedReason: string
  RequestsLastHour: number
  RequestBudget: number
}

// blockedMessage explains why a SEFAZ distribution sync is paused; time is
// the HH:MM the block ends.
export function blockedMessage(info: SefazBlockInfo, time: string) {
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

export type SefazIdleInfo = { IdleDays: number }

// IDLE_WARNING_TOOLTIP explains the idle warning of a SEFAZ distribution:
// the Ambiente Nacional only generates NSUs for CNPJs that queried it in the
// last 60 days.
export const IDLE_WARNING_TOOLTIP =
  'A SEFAZ só gera NSU para o CNPJ que consultou a distribuição nos últimos 60 dias. ' +
  'Passado esse prazo, os documentos do período parado não chegam mais por aqui. ' +
  'Sincronizar agora reinicia a contagem. Consultas de outro programa com o mesmo CNPJ ' +
  'também contam, mas o nanci não as vê.'

// idleWarning is the badge text of a distribution not queried for a long
// time, or '' when it is not at risk of losing NSUs.
export function idleWarning(info: SefazIdleInfo | null) {
  if (!info?.IdleDays) return ''
  return `Sem consulta há ${info.IdleDays} dias`
}
