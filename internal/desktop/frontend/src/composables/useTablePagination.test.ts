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

  it('starts from the saved rows-per-page preference of its table', () => {
    usePreferencesStore().rowsPerPage.nfe = 50
    expect(useTablePagination('nfe').value).toEqual({
      sortBy: 'issueDate',
      descending: true,
      page: 1,
      rowsPerPage: 50,
    })
    expect(useTablePagination('cte').value.rowsPerPage).toBe(25)
  })

  it('saves a rows-per-page change for its own table only', async () => {
    const pagination = useTablePagination('cte')
    pagination.value.rowsPerPage = 100
    await nextTick()

    expect(usePreferencesStore().rowsPerPage).toEqual({ nfse: 25, nfe: 25, cte: 100 })
    expect(useTablePagination('cte').value.rowsPerPage).toBe(100)
    expect(useTablePagination('nfse').value.rowsPerPage).toBe(25)
  })
})
