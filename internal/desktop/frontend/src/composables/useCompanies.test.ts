import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { useCompanies } from './useCompanies'
import { desktopClient } from '@/platform/wails/client'
import type { PullResult } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    assignCredential: vi.fn(),
    listCompanies: vi.fn(),
    listCredentials: vi.fn(),
    pull: vi.fn(),
    resetSyncState: vi.fn(),
  },
}))

describe('useCompanies', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('reports loading for a single list load', async () => {
    let resolveList!: (value: never[]) => void
    vi.mocked(desktopClient.listCompanies).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve
      }) as ReturnType<typeof desktopClient.listCompanies>
    )

    const companies = useCompanies()
    expect(companies.loading.value).toBe(false)

    const pending = companies.loadCompanies()
    expect(companies.loading.value).toBe(true)

    resolveList([])
    await pending
    expect(companies.loading.value).toBe(false)
  })

  it('stays loading until the outer reload finishes, not the first inner load', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])

    let resolveCompanies!: (value: never[]) => void
    vi.mocked(desktopClient.listCompanies).mockReturnValue(
      new Promise((resolve) => {
        resolveCompanies = resolve
      }) as ReturnType<typeof desktopClient.listCompanies>
    )

    const companies = useCompanies()
    const pending = companies.reloadData()

    // Credentials resolve first; the slower companies load must keep it true.
    await Promise.resolve()
    await Promise.resolve()
    expect(companies.loading.value).toBe(true)

    resolveCompanies([])
    await pending
    expect(companies.loading.value).toBe(false)
  })

  it('preserves in-flight sync state across composable instances', async () => {
    let resolvePull!: (value: PullResult) => void
    vi.mocked(desktopClient.pull).mockReturnValue(
      new Promise((resolve) => {
        resolvePull = resolve
      }) as ReturnType<typeof desktopClient.pull>
    )
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])

    const firstPage = useCompanies()
    const syncPromise = firstPage.syncCompany('123')

    const remountedPage = useCompanies()
    expect(remountedPage.isSyncingCompany('123')).toBe(true)
    expect(remountedPage.syncing.value).toBe('123')

    resolvePull({
      CompanyName: 'Empresa',
      CNPJ: '123',
      CredentialLabel: 'Certificado',
      CredentialCNPJ: '123',
      ConsultationBasis: '',
      Status: 'completed',
      StopReason: 'done',
      LastProcessedNSU: 1,
      LastFoundNSU: 1,
      EmptyStreak: 0,
      DocumentsFound: 0,
      EventsFound: 0,
      DocumentsSaved: 0,
      EventsSaved: 0,
      DocumentsSkippedByPolicy: 0,
      EventsSkippedByPolicy: 0,
      Errors: 0,
      Duration: 0,
    })
    await syncPromise

    expect(remountedPage.isSyncingCompany('123')).toBe(false)
    expect(remountedPage.syncing.value).toBeNull()
  })
})
