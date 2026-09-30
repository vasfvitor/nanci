import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { useDocuments } from './useDocuments'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useDocumentsStore } from '@/stores/documents'
import type { CompanySummary, DocumentRow, ExportResult, PullResult } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    exportDANFSe: vi.fn(),
    exportDANFSeZIP: vi.fn(),
    exportDocuments: vi.fn(),
    exportXML: vi.fn(),
    listCompanies: vi.fn(),
    listDocuments: vi.fn(),
    markDocumentsViewed: vi.fn(),
    pull: vi.fn(),
    resetSyncState: vi.fn(),
  },
}))

function documentRow(chave: string, fields: Partial<DocumentRow> = {}): DocumentRow {
  return {
    ChaveAcesso: chave,
    NFSeNumber: '1',
    PrestadorCNPJ: '11222333000181',
    PrestadorName: 'Prestador',
    TomadorCNPJ: '12345678000100',
    TomadorName: 'Tomador',
    Status: 'normal',
    ServiceDescription: '',
    ...fields,
  } as DocumentRow
}

function company(fields: Partial<CompanySummary> = {}): CompanySummary {
  return {
    CNPJ: '123',
    Name: 'Empresa',
    Environment: 'producao',
    LastFoundNSU: 7,
    ...fields,
  } as CompanySummary
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useDocuments', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company()])
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([])
  })

  it('searches with the store list input', async () => {
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'
    nfse.filter.value.Competence = '2026-06'
    nfse.filter.value.Direction = 'tomada'

    await nfse.search()

    expect(desktopClient.listDocuments).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      OnlyUnread: false,
    })
  })

  it('keeps a known company, prefers the route company and falls back to the first', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      company({ CNPJ: '111' }),
      company({ CNPJ: '222' }),
    ])
    const nfse = useDocuments()

    nfse.filter.value.CNPJ = '222'
    await nfse.loadCompanies()
    expect(nfse.filter.value.CNPJ).toBe('222')

    await nfse.loadCompanies('111')
    expect(nfse.filter.value.CNPJ).toBe('111')

    nfse.filter.value.CNPJ = '999'
    await nfse.loadCompanies()
    expect(nfse.filter.value.CNPJ).toBe('111')
  })

  it('derives the ambiente and the status line from the selected company', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      company({ Environment: 'producao_restrita', LastFoundNSU: 42 }),
    ])
    const nfse = useDocuments()
    expect(nfse.ambiente.value).toBeNull()
    expect(nfse.statusLine.value).toBe('')

    await nfse.loadCompanies()

    expect(nfse.ambiente.value).toEqual({ label: 'Produção restrita', color: 'warning' })
    expect(nfse.statusLine.value).toBe('Última sincronização: nunca · NSU 42')
  })

  it('syncs through the companySync store and refreshes the list and the companies', async () => {
    const pull = deferred<PullResult>()
    vi.mocked(desktopClient.pull).mockReturnValue(pull.promise)
    const firstPage = useDocuments()
    firstPage.filter.value.CNPJ = '123'

    const syncing = firstPage.syncNFSe()
    const remountedPage = useDocuments()
    expect(remountedPage.isSyncing.value).toBe(true)
    expect(useCompanySyncStore().isSyncing('123', 'nfse')).toBe(true)
    await expect(remountedPage.syncNFSe()).resolves.toBeNull()
    await expect(remountedPage.resetSync()).resolves.toBe(false)
    expect(desktopClient.pull).toHaveBeenCalledTimes(1)
    expect(desktopClient.resetSyncState).not.toHaveBeenCalled()

    vi.mocked(desktopClient.listCompanies).mockClear()
    pull.resolve({ CNPJ: '123', Status: 'success' } as PullResult)
    await expect(syncing).resolves.toMatchObject({ Status: 'success' })

    expect(desktopClient.pull).toHaveBeenCalledWith({ CNPJ: '123', Mode: '' })
    expect(remountedPage.isSyncing.value).toBe(false)
    expect(desktopClient.listDocuments).toHaveBeenCalled()
    // One reload of the companies, for the status line.
    expect(desktopClient.listCompanies).toHaveBeenCalledTimes(1)
  })

  it('clears the sync marker and still refreshes when the pull fails', async () => {
    vi.mocked(desktopClient.pull).mockRejectedValue(new Error('boom'))
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'

    await expect(nfse.syncNFSe()).rejects.toThrow('boom')
    expect(nfse.isSyncing.value).toBe(false)
    expect(desktopClient.listDocuments).toHaveBeenCalled()
  })

  it('resets only the sync cursor and keeps the reset visible while it is pending', async () => {
    const reset = deferred<undefined>()
    vi.mocked(desktopClient.resetSyncState).mockReturnValue(reset.promise)
    const firstPage = useDocuments()
    firstPage.filter.value.CNPJ = '123'

    const resetting = firstPage.resetSync()
    const remountedPage = useDocuments()
    expect(remountedPage.isResetting.value).toBe(true)
    await expect(remountedPage.resetSync()).resolves.toBe(false)
    await expect(remountedPage.syncNFSe()).resolves.toBeNull()
    expect(desktopClient.pull).not.toHaveBeenCalled()

    vi.mocked(desktopClient.listCompanies).mockClear()
    reset.resolve(undefined)
    await expect(resetting).resolves.toBe(true)
    expect(desktopClient.listCompanies).toHaveBeenCalledTimes(1)
    expect(desktopClient.resetSyncState).toHaveBeenCalledTimes(1)
    expect(desktopClient.resetSyncState).toHaveBeenCalledWith({ CompanyCNPJ: '123' })
    expect(remountedPage.isResetting.value).toBe(false)
  })

  it('exports exactly the given chaves, incremental when asked', async () => {
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'
    nfse.filter.value.Competence = '2026-06'
    nfse.filter.value.Direction = 'tomada'

    await nfse.exportDocuments('csv', ['a', 'b'], { incremental: false })
    await nfse.exportDANFSeZIP(['c'], { incremental: true })

    expect(desktopClient.exportDocuments).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Direction: '',
      Format: 'csv',
      Incremental: false,
      ChavesAcesso: ['a', 'b'],
    })
    expect(desktopClient.exportDANFSeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Direction: '',
      Format: 'zip',
      Incremental: true,
      ChavesAcesso: ['c'],
    })
  })

  it('does not export an empty list of chaves', async () => {
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'

    await expect(nfse.exportDocuments('zip', [], { incremental: false })).resolves.toBeNull()
    await expect(nfse.exportDANFSeZIP([], { incremental: false })).resolves.toBeNull()
    expect(desktopClient.exportDocuments).not.toHaveBeenCalled()
    expect(desktopClient.exportDANFSeZIP).not.toHaveBeenCalled()
  })

  it('exports one XML or DANFSe of the listed company', async () => {
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'

    await nfse.exportXML('chave-1')
    await nfse.exportDANFSe('chave-1')

    expect(desktopClient.exportXML).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: 'chave-1' })
    expect(desktopClient.exportDANFSe).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: 'chave-1' })
  })

  it('keeps exporting visible to a second instance while an export is pending', async () => {
    const exported = deferred<ExportResult | null>()
    vi.mocked(desktopClient.exportDocuments).mockReturnValue(exported.promise)
    const firstPage = useDocuments()
    firstPage.filter.value.CNPJ = '123'

    const pending = firstPage.exportDocuments('xlsx', ['a'], { incremental: false })
    const remountedPage = useDocuments()
    expect(remountedPage.exporting.value).toBe(true)
    await expect(remountedPage.exportDocuments('xlsx', ['a'], { incremental: false })).resolves.toBeNull()
    await expect(remountedPage.exportXML('a')).resolves.toBeNull()
    expect(desktopClient.exportDocuments).toHaveBeenCalledTimes(1)
    expect(desktopClient.exportXML).not.toHaveBeenCalled()

    exported.resolve({ OutPath: 'out.xlsx', Format: 'xlsx', Incremental: false, ExportedCount: 1 })
    await pending
    expect(remountedPage.exporting.value).toBe(false)
  })

  it('clears the export marker when the export fails', async () => {
    vi.mocked(desktopClient.exportXML).mockRejectedValue(new Error('boom'))
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'

    await expect(nfse.exportXML('a')).rejects.toThrow('boom')
    expect(nfse.exporting.value).toBe(false)
  })

  it('scopes the actions to the selection, or else to the filtered rows', () => {
    const viewed = new Date('2026-09-01T00:00:00Z')
    const a = documentRow('a', { PrestadorName: 'Outra' })
    const b = documentRow('b', { PrestadorName: 'Outra', ViewedAt: viewed })
    const c = documentRow('c')
    const nfse = useDocuments()
    useDocumentsStore().documents = [a, b, c]
    nfse.filterText.value = 'outra'

    expect(nfse.scopeRows.value).toEqual([a, b])
    expect(nfse.unviewedChaves.value).toEqual(['a'])

    nfse.selected.value = [c]
    expect(nfse.scopeRows.value).toEqual([c])
    expect(nfse.unviewedChaves.value).toEqual(['c'])
  })

  it('filters the grid by accent- and case-insensitive text and back to page 1', async () => {
    const sao = documentRow('a', { TomadorName: 'São João Ltda' })
    const other = documentRow('b', { ServiceDescription: 'Consultoria contábil' })
    const nfse = useDocuments()
    useDocumentsStore().documents = [sao, other]
    nfse.pagination.value.page = 3

    nfse.filterText.value = 'SAO JOAO'
    await nextTick()
    expect(nfse.filteredRows.value).toEqual([sao])
    expect(nfse.pagination.value.page).toBe(1)

    nfse.filterText.value = 'contabil'
    expect(nfse.filteredRows.value).toEqual([other])
  })

  it('marks NFS-e viewed by company and chave only and drops their "Novo" badge in place', async () => {
    vi.mocked(desktopClient.markDocumentsViewed).mockResolvedValue(1)
    const store = useDocumentsStore()
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'
    store.documents = [documentRow('a'), documentRow('b')]
    nfse.selected.value = [store.documents[0] as DocumentRow]
    // A filter edited after the search that filled the grid must not narrow the marking.
    nfse.filter.value.Competence = '2026-06'
    nfse.filter.value.Direction = 'tomada'

    await expect(nfse.markViewed(['a'])).resolves.toEqual({ count: 1, reloadError: null })

    expect(desktopClient.markDocumentsViewed).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Direction: '',
      OnlyUnread: false,
      ChavesAcesso: ['a'],
    })
    expect(store.documents[0]?.ViewedAt).toBeInstanceOf(Date)
    expect(store.documents[1]?.ViewedAt).toBeUndefined()
    expect(nfse.selected.value).toEqual([])
    expect(desktopClient.listDocuments).not.toHaveBeenCalled()
  })

  it('searches again after marking when only unviewed NFS-e are listed', async () => {
    vi.mocked(desktopClient.markDocumentsViewed).mockResolvedValue(2)
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'
    nfse.onlyUnviewed.value = true

    await nfse.markViewed(['a', 'b'])

    expect(desktopClient.markDocumentsViewed).toHaveBeenCalledWith(
      expect.objectContaining({ OnlyUnread: false, ChavesAcesso: ['a', 'b'] })
    )
    expect(desktopClient.listDocuments).toHaveBeenCalledWith(expect.objectContaining({ OnlyUnread: true }))
  })

  it('keeps the count when the search after marking fails', async () => {
    const failure = new Error('lista indisponível')
    vi.mocked(desktopClient.markDocumentsViewed).mockResolvedValue(2)
    vi.mocked(desktopClient.listDocuments).mockRejectedValue(failure)
    const nfse = useDocuments()
    nfse.filter.value.CNPJ = '123'
    nfse.filter.value.OnlyUnread = true

    await expect(nfse.markViewed(['a', 'b'])).resolves.toEqual({ count: 2, reloadError: failure })
    expect(nfse.markingViewed.value).toBe(false)
  })

  it('keeps "Marcar vistos" visible to a second instance while it is pending', async () => {
    const mark = deferred<number>()
    vi.mocked(desktopClient.markDocumentsViewed).mockReturnValue(mark.promise)
    const firstPage = useDocuments()
    firstPage.filter.value.CNPJ = '123'

    const marking = firstPage.markViewed(['a'])
    const remountedPage = useDocuments()
    expect(remountedPage.markingViewed.value).toBe(true)
    await expect(remountedPage.markViewed(['a'])).resolves.toBeNull()
    expect(desktopClient.markDocumentsViewed).toHaveBeenCalledTimes(1)

    mark.resolve(1)
    await marking
    expect(remountedPage.markingViewed.value).toBe(false)
  })

  it('keeps loading visible to a second instance while a search is pending', async () => {
    const list = deferred<DocumentRow[]>()
    vi.mocked(desktopClient.listDocuments).mockReturnValue(list.promise)
    const firstPage = useDocuments()
    firstPage.filter.value.CNPJ = '123'

    const pending = firstPage.search()
    expect(useDocuments().loading.value).toBe(true)

    list.resolve([])
    await pending
    expect(useDocuments().loading.value).toBe(false)
  })
})
