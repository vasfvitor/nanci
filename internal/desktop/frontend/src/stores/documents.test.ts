import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect } from 'vitest'
import { useDocumentsStore } from './documents'

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

    store.setDocuments([
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
        FirstSeenNSUValid: true,
        LastSeenNSUValid: true,
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
})
