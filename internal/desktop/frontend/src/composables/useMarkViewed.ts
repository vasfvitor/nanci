import { computed, type Ref } from 'vue'
import { withViewed } from '@/utils/formatters'

type ViewedRow = { ChaveAcesso: string; ViewedAt?: string | Date | null | undefined }

export type MarkViewedOptions<Row extends ViewedRow> = {
  // cnpj returns the company the grid shows.
  cnpj: () => string
  // filter is the page filter; OnlyUnread is the "Somente não vistos" toggle.
  filter: Ref<{ OnlyUnread?: boolean }>
  // mark sends the request for the company and the chaves only: the other
  // filters may have changed since the grid was filled.
  mark: (cnpj: string, chavesAcesso: string[]) => Promise<number>
  rows: Ref<Row[]>
  // filteredRows are the rows the grid shows.
  filteredRows: Ref<Row[]>
  selected: Ref<Row[]>
  setRows: (rows: Row[]) => void
  // markingViewed lives in the page store, so a remounted page sees the
  // request in flight.
  markingViewed: Ref<boolean>
}

// useMarkViewed holds "Marcar vistos" and "Somente não vistos" for a document
// page. With the toggle on the marked documents leave the list; otherwise
// they lose the "Novo" badge in place. The selection is cleared either way.
export function useMarkViewed<Row extends ViewedRow>(options: MarkViewedOptions<Row>) {
  const onlyUnviewed = computed({
    get: () => Boolean(options.filter.value.OnlyUnread),
    set: (value: boolean) => {
      options.filter.value.OnlyUnread = value
    },
  })

  // scopeRows are the documents "Marcar vistos" and "Exportar" act on: the
  // selection, or else every row the grid shows.
  const scopeRows = computed(() =>
    options.selected.value.length > 0 ? options.selected.value : options.filteredRows.value
  )
  const unviewedChaves = computed(() =>
    scopeRows.value.filter((row) => !row.ViewedAt).map((row) => row.ChaveAcesso)
  )

  // markViewed returns how many of the documents were new, or null when
  // nothing was sent.
  async function markViewed(chavesAcesso: string[]): Promise<number | null> {
    const cnpj = options.cnpj()
    if (!cnpj || options.markingViewed.value || chavesAcesso.length === 0) return null
    options.markingViewed.value = true
    try {
      const count = await options.mark(cnpj, chavesAcesso)
      options.selected.value = []
      const rows = options.rows.value
      if (onlyUnviewed.value) {
        const marked = new Set(chavesAcesso)
        options.setRows(rows.filter((row) => !marked.has(row.ChaveAcesso)))
      } else {
        options.setRows(withViewed(rows, chavesAcesso))
      }
      return count
    } finally {
      options.markingViewed.value = false
    }
  }

  return { onlyUnviewed, scopeRows, unviewedChaves, markViewed }
}
