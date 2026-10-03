import { computed, watch, type Ref } from 'vue'
import { normalizeText } from '@/utils/formatters'

export type RowTextFilterOptions<Row extends object> = {
  rows: Readonly<Ref<Row[]>>
  // filterText is the typed text; a cleared input sets null.
  filterText: Readonly<Ref<string | null>>
  // fields are the values of a row the text is searched in.
  fields: (row: Row) => unknown[]
  // pagination goes back to the first page when the text changes.
  pagination: Ref<{ page: number }>
}

// useRowTextFilter narrows rows to those with a field containing filterText,
// ignoring case and accents.
export function useRowTextFilter<Row extends object>(options: RowTextFilterOptions<Row>) {
  // normalized keeps the searchable fields of each row object, so a row is
  // normalized once even when the list is replaced by one that reuses it,
  // as after marking documents viewed.
  const normalized = new WeakMap<Row, string[]>()
  function searchFields(row: Row) {
    let fields = normalized.get(row)
    if (!fields) {
      fields = options.fields(row).map(normalizeText)
      normalized.set(row, fields)
    }
    return fields
  }

  const filteredRows = computed(() => {
    const query = normalizeText(options.filterText.value)
    if (!query) return options.rows.value
    return options.rows.value.filter((row) =>
      searchFields(row).some((field) => field.includes(query))
    )
  })

  watch(
    () => options.filterText.value,
    () => {
      options.pagination.value.page = 1
    }
  )

  return { filteredRows }
}
