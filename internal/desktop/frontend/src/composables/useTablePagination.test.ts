import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useTablePagination } from './useTablePagination'
import { usePreferencesStore } from '@/stores/preferences'

describe('useTablePagination', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('starts from the saved rows-per-page preference', () => {
    usePreferencesStore().rowsPerPage = 50
    expect(useTablePagination().value).toEqual({
      sortBy: 'issueDate',
      descending: true,
      page: 1,
      rowsPerPage: 50,
    })
  })

  it('saves a rows-per-page change for every table', async () => {
    const pagination = useTablePagination()
    pagination.value.rowsPerPage = 100
    await nextTick()

    expect(usePreferencesStore().rowsPerPage).toBe(100)
    expect(useTablePagination().value.rowsPerPage).toBe(100)
  })
})
