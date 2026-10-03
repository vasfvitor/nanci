import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { useQuery } from './useQuery'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn(),
    listDocuments: vi.fn().mockResolvedValue([]),
    queryNFSeEvents: vi.fn(),
  },
}))

describe('useQuery', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads the document options of the workspace company', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { ID: '1', CNPJ: '111', Name: 'Alpha' },
      { ID: '2', CNPJ: '222', Name: 'Beta' },
    ] as never)
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([
      { ChaveAcesso: '3'.repeat(50), ServiceValue: 100, CompanyRole: 'prestada', TomadorName: 'Tomador' },
    ] as never)

    const query = useQuery()
    const workspace = useWorkspaceStore()
    await workspace.loadCompanies()
    await flushPromises()

    expect(query.cnpj.value).toBe('111')
    expect(query.selectedCompany.value?.Name).toBe('Alpha')
    expect(desktopClient.listDocuments).toHaveBeenCalledWith({
      CNPJ: '111',
      Competence: '',
      Direction: '',
      OnlyUnread: false,
    })
    expect(query.documentOptions.value.map((option) => option.value)).toEqual(['3'.repeat(50)])

    // Another workspace company reloads the options and keeps the chave.
    query.form.value.chave = '3'.repeat(50)
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([])
    workspace.cnpj = '222'
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ CNPJ: '222' }))
    expect(query.documentOptions.value).toEqual([])
    expect(query.form.value.chave).toBe('3'.repeat(50))
  })

  it('calls the typed NFSe events query path', async () => {
    vi.mocked(desktopClient.queryNFSeEvents).mockResolvedValue('{"events":[]}')

    const query = useQuery()
    useWorkspaceStore().cnpj = '123'
    query.form.value = { chave: '2'.repeat(50) }

    await expect(query.runQuery()).resolves.toBe('{"events":[]}')
    expect(desktopClient.queryNFSeEvents).toHaveBeenCalledWith({
      CompanyCNPJ: '123',
      ChaveAcesso: '2'.repeat(50),
    })
  })

  it('rejects non-digit access keys before calling Wails', async () => {
    const query = useQuery()
    useWorkspaceStore().cnpj = '123'
    query.form.value = { chave: `${'1'.repeat(49)}/` }

    await expect(query.runQuery()).resolves.toBe('')

    expect(desktopClient.queryNFSeEvents).not.toHaveBeenCalled()
  })

  it('does not query without a workspace company', async () => {
    const query = useQuery()
    query.form.value = { chave: '2'.repeat(50) }

    await expect(query.runQuery()).resolves.toBe('')

    expect(desktopClient.queryNFSeEvents).not.toHaveBeenCalled()
  })

  it('ignores a superseded document fetch when the CNPJ changes again', async () => {
    const documentFor = (chave: string) => ({
      ChaveAcesso: chave,
      ServiceValue: 100,
      CompanyRole: 'prestada',
      TomadorName: 'Tomador',
      PrestadorName: 'Prestador',
    })

    let resolveSlow!: (value: unknown) => void
    vi.mocked(desktopClient.listDocuments)
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveSlow = resolve
        }) as never
      )
      .mockResolvedValueOnce([documentFor('2'.repeat(50))] as never)

    const query = useQuery()
    const workspace = useWorkspaceStore()

    workspace.cnpj = '111'
    await nextTick()
    workspace.cnpj = '222'
    await nextTick()
    await flushPromises()

    expect(query.documentOptions.value.map((option) => option.value)).toEqual(['2'.repeat(50)])

    // The first request finally lands, after its CNPJ was already replaced.
    resolveSlow([documentFor('1'.repeat(50))])
    await flushPromises()

    expect(query.documentOptions.value.map((option) => option.value)).toEqual(['2'.repeat(50)])
  })

  it('keeps loading state across composable instances while a query is pending', async () => {
    let resolveQuery!: (value: string) => void
    vi.mocked(desktopClient.queryNFSeEvents).mockReturnValue(
      new Promise<string>((resolve) => {
        resolveQuery = resolve
      })
    )

    const first = useQuery()
    useWorkspaceStore().cnpj = '123'
    first.form.value = { chave: '1'.repeat(50) }

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
