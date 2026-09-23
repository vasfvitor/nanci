import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeDocuments } from './useNFeDocuments'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeResetResult, NFeStatusResult, PullNFeResult } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
    listNFe: vi.fn(),
    listPendingManifestations: vi.fn(),
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
    Environment: 'homologacao',
    TpAmb: '2',
    AmbienteLabel: 'Homologação',
    LastCheckedNSU: 0,
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

describe('useNFeDocuments', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listNFe).mockResolvedValue([])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(status())
    vi.mocked(desktopClient.listPendingManifestations).mockResolvedValue([])
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
    expect(desktopClient.listPendingManifestations).toHaveBeenCalledWith('123')
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
})
