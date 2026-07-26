import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const defaultRowsPerPage = 25

function readRowsPerPage() {
  const stored = localStorage.getItem('nanci:documents:rowsPerPage')
  if (stored === null) return defaultRowsPerPage

  const parsed = Number(stored)
  if (!Number.isInteger(parsed) || parsed < 0) return defaultRowsPerPage

  return parsed
}

export const usePreferencesStore = defineStore('preferences', () => {
  const savedDark = localStorage.getItem('darkMode')
  let initialDark: boolean | 'auto' = 'auto'
  if (savedDark === 'true') {
    initialDark = true
  } else if (savedDark === 'false') {
    initialDark = false
  }

  const darkMode = ref<boolean | 'auto'>(initialDark)
  const rowsPerPage = ref<number>(readRowsPerPage())

  watch(darkMode, (val) => {
    localStorage.setItem('darkMode', String(val))
  })

  watch(rowsPerPage, (val) => {
    localStorage.setItem('nanci:documents:rowsPerPage', String(val))
  })

  return {
    darkMode,
    rowsPerPage
  }
})
