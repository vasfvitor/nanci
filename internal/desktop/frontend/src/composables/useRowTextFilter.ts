import { computed, watch, type Ref } from 'vue'
import { normalizeText } from '@/utils/formatters'

export type RowTextFilterOptions<Row> = {
  rows: Readonly<Ref<Row[]>>
  // filterText is the typed text; a cleared input sets null.
  filterText: Readonly<Ref<string | null>>
  // fields are the values of a row the text is searched in.
  fields: (row: Row) => unknown[]
  // onChange runs when the text changes, e.g. to go back to the first page.
  onChange?: () => void
}

// useRowTextFilter narrows rows to those with a field containing filterText,
// ignoring case and accents.
export function useRowTextFilter<Row>(options: RowTextFilterOptions<Row>) {
  // searchIndex normalizes the searchable fields once per result set, not on
  // every keystroke.
  const searchIndex = computed(() =>
    options.rows.value.map((row) => ({ row, fields: options.fields(row).map(normalizeText) }))
  )

  const filteredRows = computed(() => {
    const query = normalizeText(options.filterText.value)
    if (!query) return options.rows.value
    return searchIndex.value
      .filter(({ fields }) => fields.some((field) => field.includes(query)))
      .map(({ row }) => row)
  })

  const onChange = options.onChange
  if (onChange) {
    watch(
      () => options.filterText.value,
      () => onChange()
    )
  }

  return { filteredRows }
}
