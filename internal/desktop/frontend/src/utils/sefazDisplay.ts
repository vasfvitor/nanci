// Display helpers shared by the SEFAZ document screens (NF-e and CT-e).

type Display = { label: string; color: string }

export type FilterOption = { label: string; value: string }

// displayTable looks values up in table. An unknown value is shown as
// unknownLabel(value) in grey, and an empty one as "Desconhecido".
export function displayTable(
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
