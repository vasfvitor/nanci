import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect } from 'vitest'
import { useDocumentsStore } from './documents'
import { useWorkspaceStore } from './workspace'
import { mapDocumentRow } from '@/platform/wails/client'
import type { DocumentRow } from '@/types/desktop'

function documentRow(chave: string, overrides: Partial<DocumentRow> = {}): DocumentRow {
  return mapDocumentRow({ ID: `doc-${chave}`, ChaveAcesso: chave, Status: 'normal', ...overrides })
}

describe('documents store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('initializes with empty filter and document rows', () => {
    const store = useDocumentsStore()

    expect(store.filter).toEqual({ Direction: '', OnlyUnread: false })
    expect(store.rows).toEqual([])
  })

  it('sets the document rows', () => {
    const store = useDocumentsStore()

    store.setRows([documentRow('a')])

    expect(store.rows.map((row) => row.ChaveAcesso)).toEqual(['a'])
  })

  it('keeps filter requests mutable by feature composables', () => {
    const store = useDocumentsStore()
    store.filter.Direction = 'tomada'

    expect(store.filter).toEqual({ Direction: 'tomada', OnlyUnread: false })
  })

  it('builds the list request from the workspace and the filter', () => {
    const workspace = useWorkspaceStore()
    workspace.cnpj = '123'
    workspace.competence = '2026-06'
    const store = useDocumentsStore()
    store.filter.Direction = 'tomada'

    expect(store.listInput).toEqual({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      OnlyUnread: false,
    })
  })

  it('builds the list request from the filter, normalizing cleared fields', () => {
    const workspace = useWorkspaceStore()
    workspace.cnpj = '123'
    const store = useDocumentsStore()
    store.filter.Direction = null as unknown as string

    expect(store.listInput).toEqual({
      CNPJ: '123',
      Competence: '',
      Direction: '',
      OnlyUnread: false,
    })
  })

  it('keeps only the selected documents still present, as their fresh rows', () => {
    const store = useDocumentsStore()
    store.setRows([documentRow('a'), documentRow('b')])
    store.selected = [documentRow('a'), documentRow('b')]

    const freshA = documentRow('a', { Status: 'cancelada' })
    store.setRows([freshA, documentRow('c')])

    expect(store.selected).toEqual([freshA])
  })

  it('clears the rows, the selection and their search key', () => {
    const store = useDocumentsStore()
    store.setRows([documentRow('a')])
    store.selected = [documentRow('a')]
    store.rowsFor = { cnpj: '123', competence: '2026-05' }

    store.clearRows()

    expect(store.rows).toEqual([])
    expect(store.selected).toEqual([])
    expect(store.rowsFor).toBeNull()
  })

  it('starts with an empty text filter and no action in flight', () => {
    const store = useDocumentsStore()
    expect(store.filterText).toBe('')
    expect(store.markingViewed).toBe(false)
    expect(store.resettingCNPJ).toBe('')
  })
})
