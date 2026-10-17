import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { usePreferencesStore } from './preferences'

describe('preferences store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.restoreAllMocks()
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

  it('remembers the export format per source', () => {
    localStorage.setItem('nanci:cte:exportFormat', 'dacte-zip')
    const store = usePreferencesStore()
    expect(store.exportFormat).toEqual({ nfse: '', nfe: '', cte: 'dacte-zip' })

    store.setExportFormat('nfe', 'xlsx')

    expect(store.exportFormat).toEqual({ nfse: '', nfe: 'xlsx', cte: 'dacte-zip' })
    expect(localStorage.getItem('nanci:nfe:exportFormat')).toBe('xlsx')
    expect(localStorage.getItem('nanci:nfse:exportFormat')).toBeNull()

    setActivePinia(createPinia())
    expect(usePreferencesStore().exportFormat.nfe).toBe('xlsx')
  })

  it('falls back to defaults when storage throws', async () => {
    localStorage.setItem('darkMode', 'true')
    localStorage.setItem('nanci:nfse:rowsPerPage', '50')
    vi.spyOn(globalThis, 'localStorage', 'get').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })

    const store = usePreferencesStore()
    expect(store.darkMode).toBe('auto')
    expect(store.rowsPerPage).toEqual({ nfse: 25, nfe: 25, cte: 25 })
    expect(store.exportFormat).toEqual({ nfse: '', nfe: '', cte: '' })

    // The choices still hold for the session.
    store.darkMode = true
    store.rowsPerPage.nfe = 100
    store.setExportFormat('nfe', 'csv')
    await Promise.resolve()
    expect(store.darkMode).toBe(true)
    expect(store.rowsPerPage.nfe).toBe(100)
    expect(store.exportFormat.nfe).toBe('csv')
  })
})
