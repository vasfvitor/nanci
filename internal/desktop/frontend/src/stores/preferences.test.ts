import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { usePreferencesStore } from './preferences'

describe('preferences store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('hydrates initial values from localStorage', () => {
    localStorage.setItem('darkMode', 'false')
    localStorage.setItem('nanci:nfse:rowsPerPage', '50')
    localStorage.setItem('nanci:nfe:rowsPerPage', '100')
    localStorage.setItem('nanci:cte:rowsPerPage', '10')

    const store = usePreferencesStore()

    expect(store.darkMode).toBe(false)
    expect(store.rowsPerPage).toEqual({ nfse: 50, nfe: 100, cte: 10 })
  })

  it('falls back to the legacy shared key for a table without its own', () => {
    localStorage.setItem('nanci:documents:rowsPerPage', '50')
    localStorage.setItem('nanci:nfe:rowsPerPage', '100')

    expect(usePreferencesStore().rowsPerPage).toEqual({ nfse: 50, nfe: 100, cte: 50 })
  })

  it('hydrates the "Todos" rows per page option instead of falling back', () => {
    localStorage.setItem('nanci:documents:rowsPerPage', '50')
    localStorage.setItem('nanci:cte:rowsPerPage', '0')

    expect(usePreferencesStore().rowsPerPage.cte).toBe(0)
  })

  it('falls back to the default when rows per page is missing or invalid', () => {
    expect(usePreferencesStore().rowsPerPage).toEqual({ nfse: 25, nfe: 25, cte: 25 })

    localStorage.setItem('nanci:documents:rowsPerPage', 'nonsense')
    localStorage.setItem('nanci:nfe:rowsPerPage', '-1')
    setActivePinia(createPinia())

    expect(usePreferencesStore().rowsPerPage).toEqual({ nfse: 25, nfe: 25, cte: 25 })
  })

  it('persists dark mode and each table rows per page under its own key', async () => {
    localStorage.setItem('nanci:documents:rowsPerPage', '50')
    const store = usePreferencesStore()

    store.darkMode = true
    store.rowsPerPage.nfe = 100
    await Promise.resolve()

    expect(localStorage.getItem('darkMode')).toBe('true')
    expect(localStorage.getItem('nanci:nfe:rowsPerPage')).toBe('100')
    expect(localStorage.getItem('nanci:nfse:rowsPerPage')).toBeNull()
    expect(localStorage.getItem('nanci:cte:rowsPerPage')).toBeNull()
    expect(localStorage.getItem('nanci:documents:rowsPerPage')).toBe('50')
  })
})
