import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { useCTeDocuments } from './useCTeDocuments'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import type {
  CompanySummary,
  CTeResetResult,
  CTeRow,
  CTeStatusResult,
  ExportResult,
  PullCTeResult,
} from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
    pullCTe: vi.fn(),
    previewResetCTe: vi.fn(),
    resetCTe: vi.fn(),
    exportCTeXML: vi.fn(),
    exportCTeZIP: vi.fn(),
  },
}))

function status(overrides: Partial<CTeStatusResult> = {}): CTeStatusResult {
  return {
    CompanyName: 'Empresa',
    CNPJ: '123',
    UF: 'SP',
    TpAmb: '2',
    LastNSU: 0,
    MaxNSU: null,
    LastRunStatus: '',
    LastRunStopReason: '',
    NextAllowedAt: null,
    BlockedReason: '',
    RequestsLastHour: 0,
    RequestBudget: 20,
    TotalTomador: 0,
    TotalDestinatario: 0,
    TotalRemetente: 0,
    TotalOutros: 0,
    ...overrides,
  }
}

function cteRow(chave: string, fields: Partial<CTeRow> = {}): CTeRow {
  return {
    ChaveAcesso: chave,
    Numero: '1',
    EmitenteCNPJ: '11222333000181',
    EmitenteName: 'Transportadora',
    TomadorCNPJ: '12345678000100',
    TomadorName: 'Tomador',
    ...fields,
  } as CTeRow
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useCTeDocuments', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCTe).mockResolvedValue([])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(status())
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('searches with the store list input', async () => {
    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'
    cte.filter.value.Competence = '2026-08'
    cte.filter.value.Modelo = '57'
    cte.filter.value.Role = 'tomador'

    await cte.search()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(useCTeDocumentsStore().listInput)
    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: '123', Competence: '2026-08', Modelo: '57', Role: 'tomador' })
    )
  })

  it('does not search without a company', async () => {
    const cte = useCTeDocuments()
    await expect(cte.search()).resolves.toEqual([])
    expect(desktopClient.listCTe).not.toHaveBeenCalled()
  })

  it('explains an NF-e key filter that is not 44 characters', () => {
    const cte = useCTeDocuments()
    expect(cte.nfeChaveError.value).toBe('')

    cte.filter.value.NFeChave = '3526 0911'
    expect(cte.nfeChaveError.value).toBe('A chave de NF-e tem 44 caracteres')

    cte.filter.value.NFeChave = '3526 '.repeat(11)
    expect(cte.nfeChaveError.value).toBe('')

    // An alphanumeric CNPJ puts letters in the key.
    cte.filter.value.NFeChave = `3526 09AB ${'1234 '.repeat(9)}`
    expect(cte.nfeChaveError.value).toBe('')
  })

  it('keeps pull and export visible to a second instance while they are pending', async () => {
    const pull = deferred<PullCTeResult>()
    const zip = deferred<ExportResult | null>()
    vi.mocked(desktopClient.pullCTe).mockReturnValue(pull.promise)
    vi.mocked(desktopClient.exportCTeZIP).mockReturnValue(zip.promise)

    const firstPage = useCTeDocuments()
    firstPage.filter.value.CNPJ = '123'
    const syncing = firstPage.syncCTe()
    const exporting = firstPage.exportZIP(['chave-1'])

    const remountedPage = useCTeDocuments()
    expect(remountedPage.isSyncing.value).toBe(true)
    expect(remountedPage.exporting.value).toBe(true)
    expect(useCompanySyncStore().isSyncing('123', 'cte')).toBe(true)
    expect(useCompanySyncStore().isSyncing('123', 'nfe')).toBe(false)

    await expect(remountedPage.syncCTe()).resolves.toBeNull()
    await expect(remountedPage.exportZIP(['chave-1'])).resolves.toBeNull()
    await expect(remountedPage.exportXML('chave-1')).resolves.toBeNull()
    await expect(remountedPage.resetCTe()).resolves.toBeNull()
    expect(desktopClient.pullCTe).toHaveBeenCalledTimes(1)
    expect(desktopClient.exportCTeZIP).toHaveBeenCalledTimes(1)
    expect(desktopClient.exportCTeXML).not.toHaveBeenCalled()
    expect(desktopClient.resetCTe).not.toHaveBeenCalled()

    pull.resolve({ CNPJ: '123' } as PullCTeResult)
    zip.resolve({ OutPath: 'out.zip', ExportedCount: 1 } as ExportResult)
    await Promise.all([syncing, exporting])

    expect(remountedPage.isSyncing.value).toBe(false)
    expect(remountedPage.exporting.value).toBe(false)
    expect(desktopClient.listCTe).toHaveBeenCalled()
    expect(desktopClient.statusCTe).toHaveBeenCalledWith('123')
  })

  it('keeps the CT-e reset visible to a second instance while it is pending', async () => {
    const reset = deferred<CTeResetResult>()
    vi.mocked(desktopClient.resetCTe).mockReturnValue(reset.promise)

    const firstPage = useCTeDocuments()
    firstPage.filter.value.CNPJ = '123'
    const resetting = firstPage.resetCTe()

    const remountedPage = useCTeDocuments()
    expect(remountedPage.isResetting.value).toBe(true)
    await expect(remountedPage.previewReset()).resolves.toBeNull()
    await expect(remountedPage.resetCTe()).resolves.toBeNull()
    await expect(remountedPage.syncCTe()).resolves.toBeNull()
    expect(desktopClient.previewResetCTe).not.toHaveBeenCalled()
    expect(desktopClient.resetCTe).toHaveBeenCalledTimes(1)
    expect(desktopClient.pullCTe).not.toHaveBeenCalled()

    reset.resolve({ CNPJ: '123', CompanyDocuments: 2 } as CTeResetResult)
    await expect(resetting).resolves.toMatchObject({ CompanyDocuments: 2 })

    expect(remountedPage.isResetting.value).toBe(false)
    expect(desktopClient.listCTe).toHaveBeenCalled()
    expect(desktopClient.statusCTe).toHaveBeenCalledWith('123')
  })

  it('does not preview or reset while the CT-e sync runs', async () => {
    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'
    useCompanySyncStore().startSync('123', 'cte')

    await expect(cte.previewReset()).resolves.toBeNull()
    await expect(cte.resetCTe()).resolves.toBeNull()
    expect(desktopClient.previewResetCTe).not.toHaveBeenCalled()
    expect(desktopClient.resetCTe).not.toHaveBeenCalled()
  })

  it('clears the sync marker and refreshes the status when the pull fails', async () => {
    vi.mocked(desktopClient.pullCTe).mockRejectedValue(new Error('consultas bloqueadas'))

    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'

    await expect(cte.syncCTe()).rejects.toThrow('consultas bloqueadas')
    expect(cte.isSyncing.value).toBe(false)
    expect(desktopClient.statusCTe).toHaveBeenCalledWith('123')
  })

  it('reports syncBlockedUntil only while NextAllowedAt is in the future', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-23T12:00:00Z'))

    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'
    expect(cte.syncBlockedUntil.value).toBeNull()

    vi.mocked(desktopClient.statusCTe).mockResolvedValue(
      status({ NextAllowedAt: '2026-09-23T13:00:00Z', BlockedReason: 'rate_budget', RequestsLastHour: 20 })
    )
    await cte.loadStatus()
    expect(cte.syncBlockedUntil.value?.toISOString()).toBe('2026-09-23T13:00:00.000Z')
    expect(cte.blockedText.value).toContain('Limite de consultas por hora atingido (20/20)')

    vi.advanceTimersByTime(60 * 60 * 1000 + 1000)
    expect(cte.syncBlockedUntil.value).toBeNull()
    expect(cte.blockedText.value).toBe('')
  })

  it('exports XML and ZIP for the given chaves, incremental when asked', async () => {
    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'
    cte.filter.value.Competence = '2026-08'
    cte.filter.value.Role = 'tomador'

    await cte.exportXML('chave-1')
    await cte.exportZIP(['chave-1'])
    cte.incremental.value = true
    await cte.exportZIP(['chave-2'])

    expect(desktopClient.exportCTeXML).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: 'chave-1' })
    expect(desktopClient.exportCTeZIP).toHaveBeenNthCalledWith(1, {
      CNPJ: '123',
      Competence: '',
      Role: '',
      ChavesAcesso: ['chave-1'],
      Incremental: false,
    })
    expect(desktopClient.exportCTeZIP).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ ChavesAcesso: ['chave-2'], Incremental: true })
    )
  })

  it('exports the rows the grid shows and nothing when it is empty', async () => {
    vi.mocked(desktopClient.exportCTeZIP).mockResolvedValue(null)
    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'

    await expect(cte.exportZIP()).resolves.toBeNull()
    expect(desktopClient.exportCTeZIP).not.toHaveBeenCalled()

    useCTeDocumentsStore().rows = [cteRow('a'), cteRow('b', { EmitenteName: 'Outra' })]
    cte.filterText.value = 'outra'
    await cte.exportZIP()
    expect(desktopClient.exportCTeZIP).toHaveBeenLastCalledWith(
      expect.objectContaining({ ChavesAcesso: ['b'] })
    )
  })

  it('clears the export marker when the export fails', async () => {
    vi.mocked(desktopClient.exportCTeXML).mockRejectedValue(new Error('boom'))

    const cte = useCTeDocuments()
    cte.filter.value.CNPJ = '123'

    await expect(cte.exportXML('chave-1')).rejects.toThrow('boom')
    expect(cte.exporting.value).toBe(false)
  })

  it('filters the grid by accent- and case-insensitive text and back to page 1', async () => {
    const sao = cteRow('a', { TomadorName: 'São João Ltda' })
    const other = cteRow('b', { Numero: '4321' })
    const cte = useCTeDocuments()
    useCTeDocumentsStore().rows = [sao, other]
    cte.pagination.value.page = 3

    cte.filterText.value = 'SAO JOAO'
    await nextTick()
    expect(cte.filteredRows.value).toEqual([sao])
    expect(cte.pagination.value.page).toBe(1)

    cte.filterText.value = '432'
    expect(cte.filteredRows.value).toEqual([other])
  })

  it('keeps a known company selected and falls back to the first', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { CNPJ: '111', Name: 'Primeira' },
      { CNPJ: '222', Name: 'Segunda' },
    ] as CompanySummary[])
    const cte = useCTeDocuments()

    cte.filter.value.CNPJ = '222'
    await cte.loadCompanies()
    expect(cte.filter.value.CNPJ).toBe('222')

    cte.filter.value.CNPJ = '999'
    await cte.loadCompanies()
    expect(cte.filter.value.CNPJ).toBe('111')
  })

  it('derives the company name, count and status line from the status', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { CNPJ: '123', Name: 'Empresa da lista' } as CompanySummary,
    ])
    const cte = useCTeDocuments()
    await cte.loadCompanies()
    expect(cte.companyName.value).toContain('Empresa da lista')
    expect(cte.statusLine.value).toBe('')

    vi.mocked(desktopClient.statusCTe).mockResolvedValue(
      status({ CompanyName: 'Empresa do status', LastNSU: 5, MaxNSU: 9, TotalTomador: 3, TotalRemetente: 2 })
    )
    await cte.loadStatus()
    expect(cte.companyName.value).toBe('Empresa do status')
    expect(cte.documentCount.value).toBe(5)
    expect(cte.statusLine.value).toContain('NSU 5/9 · CT-e: 5')
  })
})
