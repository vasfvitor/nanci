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

  it('persists dark mode and rows per page changes', async () => {
    const store = usePreferencesStore()

    store.darkMode = true
    store.rowsPerPage = 100
    await Promise.resolve()

    expect(localStorage.getItem('darkMode')).toBe('true')
    expect(localStorage.getItem('nanci:documents:rowsPerPage')).toBe('100')
  })
})
