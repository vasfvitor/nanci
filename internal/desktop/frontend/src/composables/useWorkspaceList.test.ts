import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, shallowRef } from 'vue'
import { useWorkspaceList } from './useWorkspaceList'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore, type WorkspaceKey } from '@/stores/workspace'
import type { CompanySummary } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listCompanies: vi.fn() },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

const um = { CNPJ: '11111111000111', Name: 'Empresa Um' } as CompanySummary
const dois = { CNPJ: '22222222000122', Name: 'Empresa Dois' } as CompanySummary

// rowsFor and statusFor stand for the page store; they outlive each mount.
// statusFor is the company of the loaded status, as NFeStatusResult.CNPJ.
const rowsFor = shallowRef<WorkspaceKey | null>(null)
const statusFor = shallowRef('')
const clearRows = vi.fn(() => {
  rowsFor.value = null
})
const loadRows = vi.fn(async () => {})
const clearCompany = vi.fn((cnpj: string) => {
  if (statusFor.value !== cnpj) statusFor.value = ''
})
const loadCompany = vi.fn(async () => {})

// mount runs the composable in its own scope, as a mounted page would;
// unmount is scope.stop.
function mount() {
  const scope = effectScope()
  const list = scope.run(() =>
    useWorkspaceList({
      rowsFor: () => rowsFor.value,
      clearRows,
      loadRows,
      clearCompany,
      loadCompany,
    })
  ) as ReturnType<typeof useWorkspaceList>
  return { list, unmount: () => scope.stop() }
}

// loaded stands for a search and a status load that filled the page for
// the current key.
function loaded() {
  const workspace = useWorkspaceStore()
  rowsFor.value = { cnpj: workspace.cnpj, competence: workspace.competence }
  statusFor.value = workspace.cnpj
}

describe('useWorkspaceList', () => {
  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    rowsFor.value = null
    statusFor.value = ''
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um, dois])
    await useWorkspaceStore().loadCompanies()
  })

  it('loads the list and the company state on mount', () => {
    mount()

    expect(loadRows).toHaveBeenCalledTimes(1)
    expect(loadCompany).toHaveBeenCalledTimes(1)
  })

  it('clears and reloads both when the company changes', async () => {
    mount()
    loaded()
    vi.clearAllMocks()

    useWorkspaceStore().cnpj = dois.CNPJ
    await nextTick()

    expect(clearRows).toHaveBeenCalledTimes(1)
    expect(clearCompany).toHaveBeenCalledWith(dois.CNPJ)
    expect(statusFor.value).toBe('')
    expect(loadRows).toHaveBeenCalledTimes(1)
    expect(loadCompany).toHaveBeenCalledTimes(1)
  })

  it('reloads only the list when only the competência changes', async () => {
    mount()
    loaded()
    vi.clearAllMocks()

    useWorkspaceStore().competence = '2026-05'
    await nextTick()
    expect(clearRows).toHaveBeenCalledTimes(1)
    expect(loadRows).toHaveBeenCalledTimes(1)

    // A second step before the list arrives is still the same company.
    useWorkspaceStore().competence = '2026-06'
    await nextTick()
    expect(loadRows).toHaveBeenCalledTimes(2)
    expect(clearCompany).not.toHaveBeenCalled()
    expect(loadCompany).not.toHaveBeenCalled()
  })

  it('keeps the rows and the status when remounted on the same key', () => {
    mount().unmount()
    loaded()
    vi.clearAllMocks()

    mount()

    expect(clearRows).not.toHaveBeenCalled()
    expect(statusFor.value).toBe(um.CNPJ)
    expect(loadRows).toHaveBeenCalledTimes(1)
    expect(loadCompany).toHaveBeenCalledTimes(1)
  })

  it('clears the rows and the status when remounted after the company changed', () => {
    mount().unmount()
    loaded()
    vi.clearAllMocks()

    useWorkspaceStore().cnpj = dois.CNPJ
    mount()

    expect(clearRows).toHaveBeenCalledTimes(1)
    expect(statusFor.value).toBe('')
    expect(loadRows).toHaveBeenCalledTimes(1)
    expect(loadCompany).toHaveBeenCalledTimes(1)
  })

  it('works without company state', async () => {
    const scope = effectScope()
    scope.run(() => useWorkspaceList({ rowsFor: () => rowsFor.value, clearRows, loadRows }))
    loaded()

    useWorkspaceStore().cnpj = dois.CNPJ
    await nextTick()

    expect(loadRows).toHaveBeenCalledTimes(2)
    scope.stop()
  })

  it('names the workspace company and competência', async () => {
    const { list } = mount()
    expect(list.contextLine.value).toBe('Empresa Um · 11.111.111/0001-11 · Todas as competências')

    useWorkspaceStore().cnpj = dois.CNPJ
    useWorkspaceStore().competence = '2026-05'
    await nextTick()
    expect(list.contextLine.value).toBe('Empresa Dois · 22.222.222/0001-22 · Competência 05/2026')
  })

  it('does not load without a company', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()

    const { list } = mount()

    expect(loadRows).not.toHaveBeenCalled()
    expect(loadCompany).not.toHaveBeenCalled()
    expect(list.noCompanyLabel.value).toBe('Nenhuma empresa cadastrada. Cadastre uma em Empresas.')
    expect(list.contextLine.value).toBe('')
  })

  it('says why there is no company', async () => {
    setActivePinia(createPinia())
    const { list } = mount()
    expect(list.noCompanyLabel.value).toBe('Carregando empresas…')

    vi.mocked(desktopClient.listCompanies).mockRejectedValue(new Error('boom'))
    await expect(useWorkspaceStore().loadCompanies()).rejects.toThrow('boom')
    expect(list.noCompanyLabel.value).toBe('Não foi possível carregar as empresas.')
  })
})
