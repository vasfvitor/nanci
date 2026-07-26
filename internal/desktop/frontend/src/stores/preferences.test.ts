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
    localStorage.setItem('nanci:documents:rowsPerPage', '50')

    const store = usePreferencesStore()

    expect(store.darkMode).toBe(false)
    expect(store.rowsPerPage).toBe(50)
  })

  it('hydrates the "Todos" rows per page option instead of falling back', () => {
    localStorage.setItem('nanci:documents:rowsPerPage', '0')

    expect(usePreferencesStore().rowsPerPage).toBe(0)
  })

  it('falls back to the default when rows per page is missing or invalid', () => {
    expect(usePreferencesStore().rowsPerPage).toBe(25)

    localStorage.setItem('nanci:documents:rowsPerPage', 'nonsense')
    setActivePinia(createPinia())

    expect(usePreferencesStore().rowsPerPage).toBe(25)
  })

  it('persists dark mode and rows per page changes', async () => {
    const store = usePreferencesStore()

    store.darkMode = true
    store.rowsPerPage = 100
    await Promise.resolve()

    expect(localStorage.getItem('darkMode')).toBe('true')
    expect(localStorage.getItem('nanci:documents:rowsPerPage')).toBe('100')
  })
})
