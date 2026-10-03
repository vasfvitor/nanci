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

// rowsFor stands for the page store; it outlives each mount.
const rowsFor = shallowRef<WorkspaceKey | null>(null)
const clear = vi.fn(() => {
  rowsFor.value = null
})
const reload = vi.fn(async () => {})

// mount runs the composable in its own scope, as a mounted page would;
// unmount is scope.stop.
function mount() {
  const scope = effectScope()
  const list = scope.run(() =>
    useWorkspaceList({ rowsFor: () => rowsFor.value, clear, reload })
  ) as ReturnType<typeof useWorkspaceList>
  return { list, unmount: () => scope.stop() }
}

// loaded stands for a search that filled the rows for the current key.
function loaded() {
  const workspace = useWorkspaceStore()
  rowsFor.value = { cnpj: workspace.cnpj, competence: workspace.competence }
}

describe('useWorkspaceList', () => {
  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    rowsFor.value = null
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um, dois])
    await useWorkspaceStore().loadCompanies()
  })

  it('loads the list and the status on mount', () => {
    mount()

    expect(reload).toHaveBeenCalledTimes(1)
    expect(reload).toHaveBeenCalledWith({ company: true })
  })

  it('clears and reloads when the company changes', async () => {
    mount()
    loaded()
    vi.clearAllMocks()

    useWorkspaceStore().cnpj = dois.CNPJ
    await nextTick()

    expect(clear).toHaveBeenCalledWith({ company: true })
    expect(reload).toHaveBeenCalledWith({ company: true })
  })

  it('reloads only the list when only the competência changes', async () => {
    mount()
    loaded()
    vi.clearAllMocks()

    useWorkspaceStore().competence = '2026-05'
    await nextTick()
    expect(clear).toHaveBeenCalledWith({ company: false })
    expect(reload).toHaveBeenCalledWith({ company: false })

    // A second step before the list arrives is still the same company.
    useWorkspaceStore().competence = '2026-06'
    await nextTick()
    expect(reload).toHaveBeenLastCalledWith({ company: false })
  })

  it('keeps the rows when remounted on the same key', () => {
    mount().unmount()
    loaded()
    vi.clearAllMocks()

    mount()

    expect(clear).not.toHaveBeenCalled()
    expect(reload).toHaveBeenCalledWith({ company: true })
  })

  it('clears the rows when remounted after the company changed', () => {
    mount().unmount()
    loaded()
    vi.clearAllMocks()

    useWorkspaceStore().cnpj = dois.CNPJ
    mount()

    expect(clear).toHaveBeenCalledWith({ company: true })
    expect(reload).toHaveBeenCalledWith({ company: true })
  })

  it('names the workspace company and competência', async () => {
    const { list } = mount()
    expect(list.contextLine.value).toBe('Empresa Um · 11.111.111/0001-11 · Todas as competências')

    useWorkspaceStore().cnpj = dois.CNPJ
    useWorkspaceStore().competence = '2026-05'
    await nextTick()
    expect(list.contextLine.value).toBe('Empresa Dois · 22.222.222/0001-22 · Competência 05/2026')
  })

  it('does not reload without a company', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()

    const { list } = mount()

    expect(reload).not.toHaveBeenCalled()
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
