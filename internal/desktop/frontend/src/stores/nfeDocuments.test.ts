import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useNFeDocumentsStore } from './nfeDocuments'
import type { NFeRow } from '@/types/desktop'

function nfeRow(chave: string, overrides: Partial<NFeRow> = {}): NFeRow {
  return {
    ID: `rel-${chave}`,
    DocumentID: `doc-${chave}`,
    ChaveAcesso: chave,
    Serie: '1',
    Numero: '1',
    Protocolo: '',
    TipoOperacao: '1',
    EmitenteCNPJ: '12345678000199',
    EmitenteName: 'Fornecedor',
    EmitenteIE: '',
    DestinatarioCNPJ: '',
    DestinatarioName: '',
    TotalValue: 100,
    Situacao: 'autorizada',
    Completeness: 'resumo',
    Manifestacao: 'nenhuma',
    CompanyRole: 'destinatario',
    EventCount: 0,
    ...overrides,
  }
}

describe('nfeDocuments store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('builds the list request from the filter, normalizing cleared fields', () => {
    const store = useNFeDocumentsStore()
    store.filter.CNPJ = '123'
    store.filter.Competence = null as unknown as string
    store.filter.Situacao = 'autorizada'
    store.filter.Role = 'destinatario'

    expect(store.listInput).toEqual({
      CNPJ: '123',
      Competence: '',
      Situacao: 'autorizada',
      Completeness: '',
      Manifestacao: '',
      Role: 'destinatario',
      EmitenteCNPJ: '',
      OnlyUnread: false,
    })
  })

  it('prunes the selection by chave and swaps in the fresh rows', () => {
    const store = useNFeDocumentsStore()
    store.setRows([nfeRow('a'), nfeRow('b')])
    store.selected = [nfeRow('a'), nfeRow('b')]

    const freshA = nfeRow('a', { Manifestacao: 'ciencia' })
    store.setRows([freshA, nfeRow('c')])

    expect(store.selected).toEqual([freshA])
  })

  it('tracks ciência and manifestação markers per chave', () => {
    const store = useNFeDocumentsStore()

    store.startCiencia('123', ['a', 'b'])
    expect(store.cienciaInFlight).toEqual({ cnpj: '123', chaves: ['a', 'b'] })
    expect(store.isChaveBusy('a')).toBe(true)
    expect(store.isChaveBusy('c')).toBe(false)

    store.finishCiencia()
    expect(store.cienciaInFlight).toBeNull()
    expect(store.isChaveBusy('a')).toBe(false)

    store.startManifestation('c', '210200')
    store.startManifestation('d', '210240')
    expect(store.manifestationInFlight).toEqual({ c: '210200', d: '210240' })
    expect(store.isChaveBusy('c')).toBe(true)

    store.finishManifestation('c')
    expect(store.manifestationInFlight).toEqual({ d: '210240' })
    expect(store.isChaveBusy('c')).toBe(false)
    expect(store.isChaveBusy('d')).toBe(true)
  })

  it('clears the selection', () => {
    const store = useNFeDocumentsStore()
    store.selected = [nfeRow('a')]
    store.clearSelection()
    expect(store.selected).toEqual([])
  })
})
