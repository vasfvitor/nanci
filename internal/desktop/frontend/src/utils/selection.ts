// pruneSelection keeps the selected rows that are still in next, swapped for
// their fresh rows, in the order they were selected. Rows match by chave.
export function pruneSelection<Row extends { ChaveAcesso: string }>(next: Row[], selected: Row[]): Row[] {
  if (selected.length === 0) return selected
  const byChave = new Map(next.map((row) => [row.ChaveAcesso, row]))
  return selected.flatMap((row) => {
    const fresh = byChave.get(row.ChaveAcesso)
    return fresh ? [fresh] : []
  })
}
