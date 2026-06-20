import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { useDocuments } from './useDocuments'
import { desktopClient } from '@/platform/wails/client'
import type { DocumentRow, ExportResult } from '@/types/desktop'

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
      OnlyUnread: false,
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
      OutPath: '',
      Incremental: false,
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
      OutPath: '',
      Incremental: false,
    })
  })

  it('keeps loading state across composable instances while a search is pending', async () => {
    let resolveSearch!: (value: DocumentRow[]) => void
    vi.mocked(desktopClient.listDocuments).mockReturnValue(
      new Promise<DocumentRow[]>((resolve) => {
        resolveSearch = resolve
      })
    )
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])

    const first = useDocuments()
    first.filter.value.CNPJ = '123'

    const pending = first.search()
    expect(first.loading.value).toBe(true)

    const second = useDocuments()
    expect(second.loading.value).toBe(true)

    resolveSearch([])
    await pending
    expect(first.loading.value).toBe(false)
    expect(second.loading.value).toBe(false)
  })

  it('keeps exporting state across composable instances while export is pending', async () => {
    let resolveExport!: (value: ExportResult | null) => void
    vi.mocked(desktopClient.exportDocuments).mockReturnValue(
      new Promise<ExportResult | null>((resolve) => {
        resolveExport = resolve
      })
    )

    const first = useDocuments()
    first.filter.value.CNPJ = '123'

    const pending = first.exportDocuments('csv')
    expect(first.exporting.value).toBe(true)

    const second = useDocuments()
    expect(second.exporting.value).toBe(true)

    // A second export call should be ignored and return undefined
    const secondPending = second.exportDocuments('csv')
    await expect(secondPending).resolves.toBeUndefined()
    expect(desktopClient.exportDocuments).toHaveBeenCalledTimes(1)

    resolveExport({ OutPath: '/path', Format: 'csv', Incremental: false, ExportedCount: 10 })
    await pending
    expect(first.exporting.value).toBe(false)
    expect(second.exporting.value).toBe(false)
  })
})
