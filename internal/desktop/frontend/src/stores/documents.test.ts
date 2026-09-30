import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect } from 'vitest'
import { useDocumentsStore } from './documents'
import type { DocumentRow } from '@/types/desktop'

function documentRow(chave: string, overrides: Partial<DocumentRow> = {}): DocumentRow {
  return {
    ID: `doc-${chave}`,
    ChaveAcesso: chave,
    Competence: '',
    PrestadorCNPJ: '',
    PrestadorName: '',
    TomadorCNPJ: '',
    TomadorName: '',
    IntermediarioCNPJ: '',
    IntermediarioName: '',
    ServiceValue: 100,
    ISSValue: 0,
    IRRFValue: 0,
    INSSValue: 0,
    PISValue: 0,
    COFINSValue: 0,
    CSLLValue: 0,
    TotalRetentions: 0,
    Status: 'normal',
    LayoutVersion: '',
    XMLPath: '',
    RawHash: '',
    ParseWarnings: [],
    NFSeNumber: '',
    ServiceDescription: '',
    RelationID: `rel-${chave}`,
    CompanyID: 'company',
    DocumentID: `document-${chave}`,
    CompanyRole: 'tomada',
    VisibilityReason: 'exact_tomador',
    FirstSeenNSU: 1,
    LastSeenNSU: 1,
    ...overrides,
  }
}

describe('documents store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with empty filter and document rows', () => {
    const store = useDocumentsStore()

    expect(store.filter).toEqual({ CNPJ: '', Competence: '', Direction: '', OnlyUnread: false })
    expect(store.documents).toEqual([])
  })

  it('sets and resets document rows', () => {
    const store = useDocumentsStore()

    store.setRows([
      {
        ID: 'doc',
        ChaveAcesso: '',
        Competence: '',
        PrestadorCNPJ: '',
        PrestadorName: '',
        TomadorCNPJ: '',
        TomadorName: '',
        IntermediarioCNPJ: '',
        IntermediarioName: '',
        ServiceValue: 100,
        ISSValue: 0,
        IRRFValue: 0,
        INSSValue: 0,
        PISValue: 0,
        COFINSValue: 0,
        CSLLValue: 0,
        TotalRetentions: 0,
        Status: 'normal',
        LayoutVersion: '',
        XMLPath: '',
        RawHash: '',
        ParseWarnings: [],
        NFSeNumber: '',
        ServiceDescription: '',
        RelationID: 'rel',
        CompanyID: 'company',
        DocumentID: 'document',
        CompanyRole: 'tomada',
        VisibilityReason: 'exact_tomador',
        FirstSeenNSU: 1,
        LastSeenNSU: 1,
      },
    ])

    expect(store.documents).toHaveLength(1)
    store.resetDocuments()
    expect(store.documents).toEqual([])
  })

  it('keeps filter requests mutable by feature composables', () => {
    const store = useDocumentsStore()
    store.filter.CNPJ = '123'
    store.filter.Competence = '2026-06'
    store.filter.Direction = 'tomada'

    expect(store.filter).toEqual({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      OnlyUnread: false,
    })
  })

  it('builds the list request from the filter, normalizing cleared fields', () => {
    const store = useDocumentsStore()
    store.filter.CNPJ = '123'
    store.filter.Competence = null as unknown as string
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
    store.resetDocuments()
    expect(store.selected).toEqual([])
  })

  it('starts with an empty text filter', () => {
    expect(useDocumentsStore().filterText).toBe('')
  })
})
