import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { useDocuments } from './useDocuments'
import { desktopClient } from '@/platform/wails/client'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    exportDANFSe: vi.fn(),
    exportDANFSeZIP: vi.fn(),
    exportDocuments: vi.fn(),
    listCompanies: vi.fn(),
    listDocuments: vi.fn(),
    listEventsForDocument: vi.fn(),
    selectExportDirectory: vi.fn(),
  },
}))

describe('useDocuments', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('handles empty company and document results', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([])

    const documents = useDocuments()
    documents.filter.value.CNPJ = '123'

    await documents.loadCompanies()
    await documents.search()

    expect(documents.companyOptions.value).toEqual([])
    expect(documents.documents.value).toEqual([])
    expect(desktopClient.listDocuments).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Direction: '',
    })
  })

  it('builds export requests from the document filter', async () => {
    const documents = useDocuments()
    documents.filter.value.CNPJ = '123'
    documents.filter.value.Competence = '2026-06'
    documents.filter.value.Direction = 'tomada'

    await documents.exportDocuments('csv')

    expect(desktopClient.exportDocuments).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'csv',
    })
  })

  it('builds single DANFSe export requests from the document filter', async () => {
    const documents = useDocuments()
    documents.filter.value.CNPJ = '123'

    await documents.exportDANFSe('chave-1')

    expect(desktopClient.exportDANFSe).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'chave-1',
    })
  })

  it('builds DANFSe ZIP export requests from the document filter', async () => {
    const documents = useDocuments()
    documents.filter.value.CNPJ = '123'
    documents.filter.value.Competence = '2026-06'
    documents.filter.value.Direction = 'tomada'

    await documents.exportDANFSeZIP()

    expect(desktopClient.exportDANFSeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'zip',
    })
  })
})
