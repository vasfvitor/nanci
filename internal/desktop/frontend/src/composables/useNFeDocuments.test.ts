import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeDocuments } from './useNFeDocuments'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { nextTick } from 'vue'
import type {
  CompanySummary,
  ExportResult,
  NFeExportResult,
  NFeResetResult,
  NFeRow,
  NFeStatusResult,
  PullNFeResult,
} from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
    listNFe: vi.fn(),
    listNFePendingManifestacoes: vi.fn(),
    statusNFe: vi.fn(),
    pullNFe: vi.fn(),
    resetNFe: vi.fn(),
    exportNFeXML: vi.fn(),
    exportNFeZIP: vi.fn(),
  },
}))

function status(overrides: Partial<NFeStatusResult> = {}): NFeStatusResult {
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
    TotalDestinatario: 0,
    TotalEmitente: 0,
    TotalOutros: 0,
    TotalResumos: 0,
    TotalCompletas: 0,
    PendingCiencia: 0,
    PendingConclusiva: 0,
    CienciaOverdue: 0,
    ...overrides,
  }
}

function nfeRow(chave: string, fields: Partial<NFeRow> = {}): NFeRow {
  return {
    ChaveAcesso: chave,
    Numero: '1',
    EmitenteCNPJ: '11222333000181',
    EmitenteName: 'Fornecedor',
    ...fields,
  } as NFeRow
}

describe('useNFeDocuments', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listNFe).mockResolvedValue([])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(status())
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockResolvedValue([])
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('searches with the store list input', async () => {
    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'
    nfe.filter.value.Competence = '2024-09'
    nfe.filter.value.Manifestacao = 'nenhuma'

    await nfe.search()

    expect(desktopClient.listNFe).toHaveBeenCalledWith(useNFeDocumentsStore().listInput)
    expect(desktopClient.listNFe).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: '123', Competence: '2024-09', Manifestacao: 'nenhuma' })
    )
  })

  it('does not search without a company', async () => {
    const nfe = useNFeDocuments()
    await expect(nfe.search()).resolves.toEqual([])
    expect(desktopClient.listNFe).not.toHaveBeenCalled()
  })

  it('keeps the NF-e sync visible to a second instance while it is pending', async () => {
    let resolvePull!: (value: PullNFeResult) => void
    vi.mocked(desktopClient.pullNFe).mockReturnValue(
      new Promise((resolve) => {
        resolvePull = resolve
      })
    )

    const firstPage = useNFeDocuments()
    firstPage.filter.value.CNPJ = '123'
    const syncing = firstPage.syncNFe()

    const remountedPage = useNFeDocuments()
    expect(remountedPage.isSyncing.value).toBe(true)
    expect(useCompanySyncStore().isSyncing('123', 'nfe')).toBe(true)
    expect(useCompanySyncStore().isSyncing('123', 'nfse')).toBe(false)
    await expect(remountedPage.syncNFe()).resolves.toBeNull()
    expect(desktopClient.pullNFe).toHaveBeenCalledTimes(1)

    resolvePull({ CNPJ: '123' } as PullNFeResult)
    await syncing

    expect(remountedPage.isSyncing.value).toBe(false)
    expect(desktopClient.listNFe).toHaveBeenCalled()
    expect(desktopClient.statusNFe).toHaveBeenCalledWith('123')
    expect(desktopClient.listNFePendingManifestacoes).toHaveBeenCalledWith('123')
  })

  it('keeps the NF-e reset visible to a second instance while it is pending', async () => {
    let resolveReset!: (value: NFeResetResult) => void
    vi.mocked(desktopClient.resetNFe).mockReturnValue(
      new Promise((resolve) => {
        resolveReset = resolve
      })
    )

    const firstPage = useNFeDocuments()
    firstPage.filter.value.CNPJ = '123'
    const resetting = firstPage.resetNFe()

    const remountedPage = useNFeDocuments()
    expect(remountedPage.isResetting.value).toBe(true)
    await expect(remountedPage.resetNFe()).resolves.toBeNull()
    await expect(remountedPage.syncNFe()).resolves.toBeNull()
    expect(desktopClient.resetNFe).toHaveBeenCalledTimes(1)
    expect(desktopClient.pullNFe).not.toHaveBeenCalled()

    resolveReset({ CNPJ: '123', CompanyDocuments: 2 } as NFeResetResult)
    await expect(resetting).resolves.toMatchObject({ CompanyDocuments: 2 })

    expect(remountedPage.isResetting.value).toBe(false)
    expect(desktopClient.listNFe).toHaveBeenCalled()
    expect(desktopClient.statusNFe).toHaveBeenCalledWith('123')
  })

  it('does not reset while the NF-e sync runs', async () => {
    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'
    useCompanySyncStore().startSync('123', 'nfe')

    await expect(nfe.resetNFe()).resolves.toBeNull()
    expect(desktopClient.resetNFe).not.toHaveBeenCalled()
  })

  it('clears the sync marker and refreshes the status when the pull fails', async () => {
    vi.mocked(desktopClient.pullNFe).mockRejectedValue(new Error('ERR_SEFAZ_BLOCKED: bloqueado'))

    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'

    await expect(nfe.syncNFe()).rejects.toThrow('ERR_SEFAZ_BLOCKED')
    expect(nfe.isSyncing.value).toBe(false)
    expect(desktopClient.statusNFe).toHaveBeenCalledWith('123')
  })

  it('reports syncBlockedUntil only while NextAllowedAt is in the future', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-23T12:00:00Z'))

    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'
    expect(nfe.syncBlockedUntil.value).toBeNull()

    vi.mocked(desktopClient.statusNFe).mockResolvedValue(
      status({ NextAllowedAt: '2026-09-23T13:00:00Z', BlockedReason: 'caught_up' })
    )
    await nfe.loadStatus()
    expect(nfe.syncBlockedUntil.value?.toISOString()).toBe('2026-09-23T13:00:00.000Z')

    vi.advanceTimersByTime(60 * 60 * 1000 + 1000)
    expect(nfe.syncBlockedUntil.value).toBeNull()

    vi.mocked(desktopClient.statusNFe).mockResolvedValue(
      status({ NextAllowedAt: '2026-09-23T10:00:00Z' })
    )
    await nfe.loadStatus()
    expect(nfe.syncBlockedUntil.value).toBeNull()
  })

  it('does not export a ZIP without chaves', async () => {
    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'

    await expect(nfe.exportZIP([])).resolves.toBeNull()
    expect(desktopClient.exportNFeZIP).not.toHaveBeenCalled()
  })

  it('exports XML and ZIP for the given chaves', async () => {
    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'
    nfe.filter.value.Competence = '2024-09'
    nfe.filter.value.Role = 'destinatario'

    await nfe.exportXML('chave-1')
    await nfe.exportZIP(['chave-1'])

    expect(desktopClient.exportNFeXML).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: 'chave-1' })
    expect(desktopClient.exportNFeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Role: '',
      ChavesAcesso: ['chave-1'],
      IncludeResumos: false,
      Incremental: false,
    })
  })

  it('exports for the company in the list input, as the grid shows it', async () => {
    const nfe = useNFeDocuments()
    // A cleared select sets null; listInput normalizes it.
    nfe.filter.value.CNPJ = null as unknown as string

    await expect(nfe.exportXML('chave-1')).resolves.toBeNull()
    await expect(nfe.exportZIP(['chave-1'])).resolves.toBeNull()
    expect(desktopClient.exportNFeXML).not.toHaveBeenCalled()
    expect(desktopClient.exportNFeZIP).not.toHaveBeenCalled()

    nfe.filter.value.CNPJ = '123'
    await nfe.exportXML('chave-1')
    await nfe.exportZIP(['chave-1'])

    const cnpj = useNFeDocumentsStore().listInput.CNPJ
    expect(desktopClient.exportNFeXML).toHaveBeenCalledWith(expect.objectContaining({ CNPJ: cnpj }))
    expect(desktopClient.exportNFeZIP).toHaveBeenCalledWith(expect.objectContaining({ CNPJ: cnpj }))
  })

  it('keeps an export visible to a second instance while it is pending', async () => {
    let resolveExport!: (value: NFeExportResult | null) => void
    vi.mocked(desktopClient.exportNFeZIP).mockReturnValue(
      new Promise((resolve) => {
        resolveExport = resolve
      })
    )

    const firstPage = useNFeDocuments()
    firstPage.filter.value.CNPJ = '123'
    const exporting = firstPage.exportZIP(['chave-1'])

    const remountedPage = useNFeDocuments()
    expect(remountedPage.exporting.value).toBe(true)
    await expect(remountedPage.exportXML('chave-1')).resolves.toBeNull()
    await expect(remountedPage.exportZIP(['chave-1'])).resolves.toBeNull()
    expect(desktopClient.exportNFeXML).not.toHaveBeenCalled()
    expect(desktopClient.exportNFeZIP).toHaveBeenCalledTimes(1)

    resolveExport({ OutPath: 'out.zip', ExportedCount: 1, SkippedResumos: 0 } as NFeExportResult)
    await exporting

    expect(remountedPage.exporting.value).toBe(false)
  })

  it('clears the export marker when the export fails', async () => {
    vi.mocked(desktopClient.exportNFeXML).mockRejectedValue(new Error('boom'))

    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'

    await expect(nfe.exportXML('chave-1')).rejects.toThrow('boom')
    expect(nfe.exporting.value).toBe(false)
    vi.mocked(desktopClient.exportNFeXML).mockResolvedValue({ OutPath: 'a.xml' } as ExportResult)
    await expect(nfe.exportXML('chave-1')).resolves.toMatchObject({ OutPath: 'a.xml' })
  })

  it('filters the grid by accent- and case-insensitive text and back to page 1', async () => {
    const sao = nfeRow('a', { EmitenteName: 'São João Ltda' })
    const other = nfeRow('b', { Numero: '4321' })
    const nfe = useNFeDocuments()
    useNFeDocumentsStore().setRows([sao, other])
    nfe.pagination.value.page = 3

    expect(nfe.filteredRows.value).toEqual([sao, other])
    nfe.filterText.value = 'SAO JOAO'
    await nextTick()
    expect(nfe.filteredRows.value).toEqual([sao])
    expect(nfe.pagination.value.page).toBe(1)

    nfe.filterText.value = '432'
    expect(nfe.filteredRows.value).toEqual([other])
  })

  it('exports the selected rows, or else the rows the grid shows', async () => {
    vi.mocked(desktopClient.exportNFeZIP).mockResolvedValue(null)
    const nfe = useNFeDocuments()
    nfe.filter.value.CNPJ = '123'
    useNFeDocumentsStore().setRows([nfeRow('a'), nfeRow('b', { EmitenteName: 'Outro' })])
    nfe.filterText.value = 'outro'

    await nfe.exportZIP()
    expect(desktopClient.exportNFeZIP).toHaveBeenLastCalledWith(
      expect.objectContaining({ ChavesAcesso: ['b'] })
    )

    nfe.selected.value = [nfeRow('a')]
    await nfe.exportZIP()
    expect(desktopClient.exportNFeZIP).toHaveBeenLastCalledWith(
      expect.objectContaining({ ChavesAcesso: ['a'] })
    )
  })

  it('keeps a known company selected and falls back to the first', async () => {
    const companies = [
      { CNPJ: '111', Name: 'Primeira' },
      { CNPJ: '222', Name: 'Segunda' },
    ] as CompanySummary[]
    vi.mocked(desktopClient.listCompanies).mockResolvedValue(companies)
    const nfe = useNFeDocuments()

    nfe.filter.value.CNPJ = '222'
    await nfe.loadCompanies()
    expect(nfe.filter.value.CNPJ).toBe('222')

    nfe.filter.value.CNPJ = '999'
    await nfe.loadCompanies()
    expect(nfe.filter.value.CNPJ).toBe('111')
  })

  it('derives the company name, counts and status line from the status', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { CNPJ: '123', Name: 'Empresa da lista' } as CompanySummary,
    ])
    const nfe = useNFeDocuments()
    await nfe.loadCompanies()
    expect(nfe.companyName.value).toContain('Empresa da lista')
    expect(nfe.statusLine.value).toBe('')

    vi.mocked(desktopClient.statusNFe).mockResolvedValue(
      status({ CompanyName: 'Empresa do status', LastNSU: 5, MaxNSU: 9, PendingCiencia: 1, PendingConclusiva: 2, TotalDestinatario: 3, TotalEmitente: 1 })
    )
    await nfe.loadStatus()
    expect(nfe.companyName.value).toBe('Empresa do status')
    expect(nfe.pendingCount.value).toBe(3)
    expect(nfe.noteCount.value).toBe(4)
    expect(nfe.statusLine.value).toContain('NSU 5/9 · Pendências: 3')
  })
})
