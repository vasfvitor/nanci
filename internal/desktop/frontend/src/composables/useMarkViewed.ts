import { computed, type Ref } from 'vue'
import { withViewed } from '@/utils/formatters'

type ViewedRow = { ChaveAcesso: string; ViewedAt?: string | Date | null | undefined }

export type MarkViewedOptions<Row extends ViewedRow> = {
  // filter is the page filter; its CNPJ is the company the grid shows and
  // OnlyUnread is the "Somente não vistos" toggle.
  filter: Ref<{ CNPJ: string; OnlyUnread?: boolean }>
  // mark sends the request for the company and the chaves only: the other
  // filters may have changed since the grid was filled.
  mark: (cnpj: string, chavesAcesso: string[]) => Promise<number>
  rows: Ref<Row[]>
  selected: Ref<Row[]>
  setRows: (rows: Row[]) => void
  search: () => Promise<unknown>
  // markingViewed lives in the page store, so a remounted page sees the
  // request in flight.
  markingViewed: Ref<boolean>
}

export type MarkViewedResult = {
  // count is how many of the documents were new.
  count: number
  // reloadError is why the list could not be searched again after the
  // marking, or null. The marking itself went through.
  reloadError: unknown
}

// useMarkViewed holds "Marcar vistos" and "Somente não vistos" for a document
// page. With the toggle on the list is searched again after marking, so the
// marked documents leave it; otherwise they lose the "Novo" badge in place.
// The selection is cleared either way.
export function useMarkViewed<Row extends ViewedRow>(options: MarkViewedOptions<Row>) {
  const onlyUnviewed = computed({
    get: () => Boolean(options.filter.value.OnlyUnread),
    set: (value: boolean) => {
      options.filter.value.OnlyUnread = value
    },
  })

  async function markViewed(chavesAcesso: string[]): Promise<MarkViewedResult | null> {
    const cnpj = options.filter.value.CNPJ
    if (!cnpj || options.markingViewed.value || chavesAcesso.length === 0) return null
    options.markingViewed.value = true
    try {
      const count = await options.mark(cnpj, chavesAcesso)
      options.selected.value = []
      let reloadError: unknown = null
      if (onlyUnviewed.value) {
        try {
          await options.search()
        } catch (error) {
          reloadError = error
        }
      } else {
        options.setRows(withViewed(options.rows.value, chavesAcesso))
      }
      return { count, reloadError }
    } finally {
      options.markingViewed.value = false
    }
  }

  return { onlyUnviewed, markViewed }
}
