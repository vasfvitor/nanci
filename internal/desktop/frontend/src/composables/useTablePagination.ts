import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePreferencesStore, type DocumentTable } from '@/stores/preferences'

// useTablePagination is the pagination of a document table, newest first.
// Each table keeps its own rows-per-page choice as a saved preference.
export function useTablePagination(table: DocumentTable) {
  const { rowsPerPage } = storeToRefs(usePreferencesStore())

  const pagination = ref({
    sortBy: 'issueDate',
    descending: true,
    page: 1,
    rowsPerPage: rowsPerPage.value[table],
  })

  watch(
    () => pagination.value.rowsPerPage,
    (value) => {
      rowsPerPage.value[table] = value
    }
  )

  return pagination
}
