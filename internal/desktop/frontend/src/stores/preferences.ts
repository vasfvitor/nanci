import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const usePreferencesStore = defineStore('preferences', () => {
  const savedDark = localStorage.getItem('darkMode')
  let initialDark: boolean | 'auto' = 'auto'
  if (savedDark === 'true') {
    initialDark = true
  } else if (savedDark === 'false') {
    initialDark = false
  }

  const darkMode = ref<boolean | 'auto'>(initialDark)
  const rowsPerPage = ref<number>(Number(localStorage.getItem('nanci:documents:rowsPerPage')) || 25)

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
