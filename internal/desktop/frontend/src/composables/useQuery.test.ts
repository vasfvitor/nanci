import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { effectScope, nextTick } from 'vue'
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
    const alphaChave = '355030812' + '12ABC34501DE35' + '000000000012326081234567897'
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([
      { ChaveAcesso: '3'.repeat(50), ServiceValue: 100, CompanyRole: 'prestada', TomadorName: 'Tomador' },
      { ChaveAcesso: alphaChave, ServiceValue: 100, CompanyRole: 'tomada', PrestadorName: 'Alfa' },
    ] as never)

    const query = useQuery()
    const workspace = useWorkspaceStore()
    await workspace.loadCompanies()
    await flushPromises()

    expect(workspace.cnpj).toBe('111')
    expect(desktopClient.listDocuments).toHaveBeenCalledWith({
      CNPJ: '111',
      Competence: '',
      Direction: '',
      OnlyUnread: false,
    })
    // The letters of an alphanumeric CNPJ stay in the chave.
    expect(query.documentOptions.value.map((option) => option.value)).toEqual([
      '3'.repeat(50),
      alphaChave,
    ])

    // Another workspace company reloads the options and keeps the chave.
    query.form.value.chave = '3'.repeat(50)
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([])
    workspace.cnpj = '222'
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ CNPJ: '222' }))
    expect(query.documentOptions.value).toEqual([])
    expect(query.form.value.chave).toBe('3'.repeat(50))
  })

  it('lists the suggestions once per company across page visits', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([
      { ID: '1', CNPJ: '111', Name: 'Alpha' },
      { ID: '2', CNPJ: '222', Name: 'Beta' },
    ] as never)
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([
      { ChaveAcesso: '3'.repeat(50), ServiceValue: 12345, CompanyRole: 'tomada', PrestadorName: 'Prestador' },
    ] as never)
    const workspace = useWorkspaceStore()
    await workspace.loadCompanies()

    // Each visit runs in its own scope; leaving the page stops it.
    const firstVisit = effectScope()
    firstVisit.run(() => useQuery())
    await flushPromises()
    expect(desktopClient.listDocuments).toHaveBeenCalledTimes(1)
    firstVisit.stop()

    // A second visit to the page, for the same company.
    const second = useQuery()
    await flushPromises()
    expect(desktopClient.listDocuments).toHaveBeenCalledTimes(1)
    expect(second.documentOptions.value).toEqual([
      {
        label: '3'.repeat(50),
        value: '3'.repeat(50),
        description: '...333333 | Prestador | R$ 123,45',
      },
    ])

    workspace.cnpj = '222'
    await flushPromises()
    expect(desktopClient.listDocuments).toHaveBeenCalledTimes(2)
    expect(desktopClient.listDocuments).toHaveBeenLastCalledWith(
      expect.objectContaining({ CNPJ: '222' })
    )
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

  it('queries a chave with an alphanumeric CNPJ, upper-cased', async () => {
    vi.mocked(desktopClient.queryNFSeEvents).mockResolvedValue('{}')
    const chave = '355030812' + '12ABC34501DE35' + '000000000012326081234567897'

    const query = useQuery()
    useWorkspaceStore().cnpj = '123'
    query.form.value = { chave: ` ${chave.toLowerCase()} ` }

    await expect(query.runQuery()).resolves.toBe('{}')
    expect(desktopClient.queryNFSeEvents).toHaveBeenCalledWith({ CompanyCNPJ: '123', ChaveAcesso: chave })
  })

  it('rejects letters outside the inscrição federal before calling Wails', async () => {
    const query = useQuery()
    useWorkspaceStore().cnpj = '123'
    query.form.value = { chave: `${'1'.repeat(49)}A` }

    await expect(query.runQuery()).resolves.toBe('')
    expect(desktopClient.queryNFSeEvents).not.toHaveBeenCalled()
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
