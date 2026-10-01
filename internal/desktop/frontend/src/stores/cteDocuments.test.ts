import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useCTeDocumentsStore } from './cteDocuments'
import type { CTeRow } from '@/types/desktop'

function cteRow(chave: string, overrides: Partial<CTeRow> = {}) {
  return { ChaveAcesso: chave, Numero: '1', ...overrides } as CTeRow
}

describe('cteDocuments store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('builds the list request from the filter, normalizing cleared fields', () => {
    const store = useCTeDocumentsStore()
    store.filter.CNPJ = '123'
    store.filter.Competence = null as unknown as string
    store.filter.Situacao = 'cancelada'
    store.filter.Role = 'remetente'
    store.filter.Modelo = '67'
    store.filter.TomadorCNPJ = null as unknown as string

    expect(store.listInput).toEqual({
      CNPJ: '123',
      Competence: '',
      Situacao: 'cancelada',
      Role: 'remetente',
      Modelo: '67',
      EmitenteCNPJ: '',
      TomadorCNPJ: '',
      NFeChave: '',
      OnlyUnread: false,
    })
  })

  it('sends "Somente não vistos" in the list request', () => {
    const store = useCTeDocumentsStore()
    store.filter.OnlyUnread = true
    expect(store.listInput.OnlyUnread).toBe(true)
  })

  it('keeps only the letters and digits of the typed CNPJ and NF-e key', () => {
    const store = useCTeDocumentsStore()
    store.filter.TomadorCNPJ = '12.abc.678/0001-00'
    store.filter.NFeChave = ' 3526 0911 2223 3300 0181 5500 1000 0045 1214 1827 3651 '

    expect(store.listInput.TomadorCNPJ).toBe('12ABC678000100')
    expect(store.listInput.NFeChave).toBe('35260911222333000181550010000045121418273651')
  })

  it('keeps only the selected CT-e still present, as their fresh rows', () => {
    const store = useCTeDocumentsStore()
    store.setRows([cteRow('a'), cteRow('b')])
    store.selected = [cteRow('a'), cteRow('b')]

    const freshB = cteRow('b', { Numero: '2' })
    store.setRows([freshB, cteRow('c')])

    expect(store.rows.map((row) => row.ChaveAcesso)).toEqual(['b', 'c'])
    expect(store.selected).toEqual([freshB])
  })

  it('starts with an empty text filter and selection', () => {
    const store = useCTeDocumentsStore()
    expect(store.filterText).toBe('')
    expect(store.selected).toEqual([])
  })

  it('flags an NF-e key filter that is not 44 characters', () => {
    const store = useCTeDocumentsStore()
    expect(store.listError).toBe('')

    store.filter.NFeChave = '3526 0911'
    expect(store.listError).toBe('A chave de NF-e tem 44 caracteres')

    // An alphanumeric CNPJ puts letters in the key.
    store.filter.NFeChave = `3526 09AB ${'1234 '.repeat(9)}`
    expect(store.listError).toBe('')
  })
})
