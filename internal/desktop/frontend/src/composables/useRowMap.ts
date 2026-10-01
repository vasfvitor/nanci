import { computed, type Ref } from 'vue'

// useRowMap builds a value for each row once per result set, keyed by
// keyOf, so a template reads it instead of rebuilding it on each render.
export function useRowMap<Row, Value>(
  rows: Readonly<Ref<Row[]>>,
  keyOf: (row: Row) => string,
  build: (row: Row) => Value
) {
  return computed(() => new Map(rows.value.map((row) => [keyOf(row), build(row)])))
}
