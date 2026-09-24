import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useCTeDocumentsStore } from './cteDocuments'

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
    })
  })

  it('keeps only the letters and digits of the typed CNPJ and NF-e key', () => {
    const store = useCTeDocumentsStore()
    store.filter.TomadorCNPJ = '12.abc.678/0001-00'
    store.filter.NFeChave = ' 3526 0911 2223 3300 0181 5500 1000 0045 1214 1827 3651 '

    expect(store.listInput.TomadorCNPJ).toBe('12ABC678000100')
    expect(store.listInput.NFeChave).toBe('35260911222333000181550010000045121418273651')
  })
})
