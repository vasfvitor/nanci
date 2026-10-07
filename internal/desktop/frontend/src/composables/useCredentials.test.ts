import { beforeEach, expect, vi } from 'vitest'
import { useCredentials } from './useCredentials'
import { desktopClient } from '@/platform/wails/client'
import { deferred } from '@/test/fixtures'
import type { CredentialSummary } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCredentials: vi.fn(),
    selectCertificate: vi.fn(),
    updateCredentialData: vi.fn(),
    updateCredentialPath: vi.fn(),
  },
}))

const credential = { ID: 'cred-1', Label: 'A1 matriz', CertPath: 'C:\\certs\\a1.pfx' } as CredentialSummary

describe('useCredentials', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([credential])
  })

  it('reports loading while the list loads and fills credentials', async () => {
    const list = deferred<CredentialSummary[]>()
    vi.mocked(desktopClient.listCredentials).mockReturnValue(list.promise)

    const api = useCredentials()
    expect(api.loading.value).toBe(false)

    const pending = api.loadCredentials()
    expect(api.loading.value).toBe(true)

    list.resolve([credential])
    await expect(pending).resolves.toEqual([credential])
    expect(api.loading.value).toBe(false)
    expect(api.credentials.value).toEqual([credential])
  })

  it('passes a list error to the caller and clears loading', async () => {
    vi.mocked(desktopClient.listCredentials).mockRejectedValue(new Error('banco indisponível'))

    const api = useCredentials()
    await expect(api.loadCredentials()).rejects.toThrow('banco indisponível')

    expect(api.loading.value).toBe(false)
    expect(api.credentials.value).toEqual([])
  })

  it('returns the chosen certificate path, or null when the dialog is cancelled', async () => {
    const api = useCredentials()

    vi.mocked(desktopClient.selectCertificate).mockResolvedValueOnce('C:\\certs\\novo.pfx')
    await expect(api.selectCertificate()).resolves.toBe('C:\\certs\\novo.pfx')

    vi.mocked(desktopClient.selectCertificate).mockResolvedValueOnce(null)
    await expect(api.selectCertificate()).resolves.toBeNull()
  })

  it('updates the certificate path and reloads the list', async () => {
    const api = useCredentials()
    await api.updateCredentialPath('cred-1', 'C:\\certs\\novo.pfx')

    expect(desktopClient.updateCredentialPath).toHaveBeenCalledWith({
      CredentialID: 'cred-1',
      CertPath: 'C:\\certs\\novo.pfx',
    })
    expect(desktopClient.listCredentials).toHaveBeenCalledOnce()
    expect(api.credentials.value).toEqual([credential])
  })

  it('updates the label and reloads the list', async () => {
    const api = useCredentials()
    await api.updateCredentialData('cred-1', 'A1 filial')

    expect(desktopClient.updateCredentialData).toHaveBeenCalledWith({ CredentialID: 'cred-1', Label: 'A1 filial' })
    expect(desktopClient.listCredentials).toHaveBeenCalledOnce()
  })

  it('does not reload the list when an update fails', async () => {
    vi.mocked(desktopClient.updateCredentialPath).mockRejectedValue(new Error('arquivo não encontrado'))
    vi.mocked(desktopClient.updateCredentialData).mockRejectedValue(new Error('nome inválido'))

    const api = useCredentials()
    await expect(api.updateCredentialPath('cred-1', 'C:\\x.pfx')).rejects.toThrow('arquivo não encontrado')
    await expect(api.updateCredentialData('cred-1', '')).rejects.toThrow('nome inválido')

    expect(desktopClient.listCredentials).not.toHaveBeenCalled()
  })

  it('keeps the list per instance, as route-local state', async () => {
    const first = useCredentials()
    await first.loadCredentials()

    const second = useCredentials()
    expect(second.credentials.value).toEqual([])
    expect(second.loading.value).toBe(false)
  })
})
