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
    vi.mocked(desktopClient.selectExportDirectory).mockResolvedValue('C:\\exports')
    vi.mocked(desktopClient.exportDocuments).mockResolvedValue({
      OutPath: 'C:\\exports\\export.csv',
      Format: 'csv',
    })

    const documents = useDocuments()
    documents.filter.value = { CNPJ: '123', Competence: '2026-06', Direction: 'tomada' }

    await documents.exportDocuments('csv')

    expect(desktopClient.exportDocuments).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'csv',
      OutDir: 'C:\\exports',
    })
  })

  it('does not export when directory selection is cancelled', async () => {
    vi.mocked(desktopClient.selectExportDirectory).mockResolvedValue(null)

    const documents = useDocuments()
    await expect(documents.exportDocuments('zip')).resolves.toBeNull()

    expect(desktopClient.exportDocuments).not.toHaveBeenCalled()
  })

  it('builds single DANFSe export requests from the document filter', async () => {
    vi.mocked(desktopClient.selectExportDirectory).mockResolvedValue('C:\\exports')
    vi.mocked(desktopClient.exportDANFSe).mockResolvedValue({
      OutPath: 'C:\\exports\\danfse.pdf',
      Format: 'danfse',
    })

    const documents = useDocuments()
    documents.filter.value = { CNPJ: '123', Competence: '2026-06', Direction: 'tomada' }

    await documents.exportDANFSe('chave-1')

    expect(desktopClient.exportDANFSe).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'chave-1',
      OutDir: 'C:\\exports',
    })
  })

  it('builds DANFSe ZIP export requests from the document filter', async () => {
    vi.mocked(desktopClient.selectExportDirectory).mockResolvedValue('C:\\exports')
    vi.mocked(desktopClient.exportDANFSeZIP).mockResolvedValue({
      OutPath: 'C:\\exports\\danfses.zip',
      Format: 'danfse-zip',
    })

    const documents = useDocuments()
    documents.filter.value = { CNPJ: '123', Competence: '2026-06', Direction: 'tomada' }

    await documents.exportDANFSeZIP()

    expect(desktopClient.exportDANFSeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'zip',
      OutDir: 'C:\\exports',
    })
  })
})
