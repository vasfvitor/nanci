import { ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { usePreferencesStore } from '@/stores/preferences'

// useTablePagination is the pagination of a document table, newest first.
// The rows-per-page choice is a saved preference shared by every table.
export function useTablePagination() {
  const { rowsPerPage } = storeToRefs(usePreferencesStore())

  const pagination = ref({
    sortBy: 'issueDate',
    descending: true,
    page: 1,
    rowsPerPage: rowsPerPage.value,
  })

  watch(
    () => pagination.value.rowsPerPage,
    (value) => {
      rowsPerPage.value = value
    }
  )

  return pagination
}
