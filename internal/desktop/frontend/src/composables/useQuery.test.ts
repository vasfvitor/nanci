import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { useQuery } from './useQuery'
import { desktopClient } from '@/platform/wails/client'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
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

  it('calls the typed NFSe events query path', async () => {
    vi.mocked(desktopClient.queryNFSeEvents).mockResolvedValue('{"events":[]}')

    const query = useQuery()
    query.form.value = { cnpj: '123', chave: '2'.repeat(50) }

    await expect(query.runQuery()).resolves.toBe('{"events":[]}')
    expect(desktopClient.queryNFSeEvents).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: '2'.repeat(50),
    })
  })

  it('rejects non-digit access keys before calling Wails', async () => {
    const query = useQuery()
    query.form.value = { cnpj: '123', chave: `${'1'.repeat(49)}/` }

    await expect(query.runQuery()).resolves.toBe('')

    expect(desktopClient.queryNFSeEvents).not.toHaveBeenCalled()
  })

  it('keeps loading state across composable instances while a query is pending', async () => {
    let resolveQuery!: (value: string) => void
    vi.mocked(desktopClient.queryNFSeEvents).mockReturnValue(
      new Promise<string>((resolve) => {
        resolveQuery = resolve
      })
    )

    const first = useQuery()
    first.form.value = { cnpj: '123', chave: '1'.repeat(50) }

    const pending = first.runQuery()
    expect(first.loading.value).toBe(true)

    const second = useQuery()
    expect(second.loading.value).toBe(true)

    await expect(second.runQuery()).resolves.toBe('')
    expect(desktopClient.queryNFSeEvents).toHaveBeenCalledTimes(1)

    resolveQuery('{"ok":true}')
    await expect(pending).resolves.toBe('{"ok":true}')
    expect(second.loading.value).toBe(false)
  })
})
