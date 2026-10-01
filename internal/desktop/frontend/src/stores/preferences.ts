import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type DocumentTable = 'nfse' | 'nfe' | 'cte'

const documentTables: DocumentTable[] = ['nfse', 'nfe', 'cte']
const defaultRowsPerPage = 25

// legacyRowsPerPageKey held one choice for every document table. It is read
// as the fallback of a table without its own key, and never removed.
const legacyRowsPerPageKey = 'nanci:documents:rowsPerPage'

function rowsPerPageKey(table: DocumentTable) {
  return `nanci:${table}:rowsPerPage`
}

// parseRowsPerPage accepts a non-negative integer; 0 is Quasar's "Todos".
function parseRowsPerPage(stored: string | null) {
  if (stored === null) return null
  const parsed = Number(stored)
  if (!Number.isInteger(parsed) || parsed < 0) return null
  return parsed
}

function readRowsPerPage(table: DocumentTable) {
  return (
    parseRowsPerPage(localStorage.getItem(rowsPerPageKey(table))) ??
    parseRowsPerPage(localStorage.getItem(legacyRowsPerPageKey)) ??
    defaultRowsPerPage
  )
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

  // rowsPerPage is the rows-per-page choice of each document table.
  const rowsPerPage = ref(
    Object.fromEntries(documentTables.map((table) => [table, readRowsPerPage(table)])) as Record<
      DocumentTable,
      number
    >
  )

  watch(darkMode, (val) => {
    localStorage.setItem('darkMode', String(val))
  })

  // A choice is saved under its table's key once it differs from what
  // storage gives, so a table keeps following the legacy key until then.
  watch(
    rowsPerPage,
    (choices) => {
      for (const table of documentTables) {
        if (choices[table] !== readRowsPerPage(table)) {
          localStorage.setItem(rowsPerPageKey(table), String(choices[table]))
        }
      }
    },
    { deep: true }
  )

  return {
    darkMode,
    rowsPerPage,
  }
})
