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
      LastCheckedNSU: 1,
      LastFoundNSU: 1,
      LastFoundNSUValid: true,
      EmptyStreak: 0,
      DocumentsFound: 0,
      EventsFound: 0,
      Errors: 0,
      Duration: 0,
    })
    await syncPromise

    expect(remountedPage.isSyncingCompany('123')).toBe(false)
    expect(remountedPage.syncing.value).toBeNull()
  })
})
