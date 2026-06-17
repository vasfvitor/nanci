import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { useQuery } from './useQuery'
import { desktopClient } from '@/platform/wails/client'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
    queryNFSe: vi.fn(),
    queryNFSeEvents: vi.fn(),
  },
}))

describe('useQuery', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads and filters company options', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { ID: '1', CNPJ: '111', Name: 'Alpha' },
      { ID: '2', CNPJ: '222', Name: 'Beta' },
    ] as never)

    const query = useQuery()
    await query.loadCompanies()
    query.filterCompanies('alp')

    expect(query.companyOptions.value).toEqual([{ label: 'Alpha (111)', value: '111' }])
  })

  it('calls the typed NFSe query path', async () => {
    vi.mocked(desktopClient.queryNFSe).mockResolvedValue('{"ok":true}')

    const query = useQuery()
    query.form.value = { cnpj: '123', chave: '1'.repeat(50) }
    query.type.value = 'nfse'

    await expect(query.runQuery()).resolves.toBe('{"ok":true}')
    expect(desktopClient.queryNFSe).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: '1'.repeat(50),
    })
    expect(desktopClient.queryNFSeEvents).not.toHaveBeenCalled()
  })

  it('calls the typed NFSe events query path', async () => {
    vi.mocked(desktopClient.queryNFSeEvents).mockResolvedValue('{"events":[]}')

    const query = useQuery()
    query.form.value = { cnpj: '123', chave: '2'.repeat(50) }
    query.type.value = 'events'

    await expect(query.runQuery()).resolves.toBe('{"events":[]}')
    expect(desktopClient.queryNFSeEvents).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: '2'.repeat(50),
    })
    expect(desktopClient.queryNFSe).not.toHaveBeenCalled()
  })
})
