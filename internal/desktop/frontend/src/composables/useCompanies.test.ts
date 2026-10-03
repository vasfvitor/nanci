import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, vi } from 'vitest'
import { companyOption, useCompanies } from './useCompanies'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CompanySummary } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    assignCredential: vi.fn(),
    listCompanies: vi.fn(),
    listCredentials: vi.fn(),
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

  it('loadCompanies fills the workspace list', async () => {
    const company = { CNPJ: '11111111000111', Name: 'Empresa Um' } as CompanySummary
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])

    const companies = useCompanies()
    await expect(companies.loadCompanies()).resolves.toEqual([company])

    const workspace = useWorkspaceStore()
    expect(workspace.companies).toEqual([company])
    expect(workspace.cnpj).toBe(company.CNPJ)
    expect(companies.companies.value).toEqual([company])
  })

  it('stays loading until both loads finish, not the first one', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])

    let resolveCompanies!: (value: never[]) => void
    vi.mocked(desktopClient.listCompanies).mockReturnValue(
      new Promise((resolve) => {
        resolveCompanies = resolve
      }) as ReturnType<typeof desktopClient.listCompanies>
    )

    const companies = useCompanies()
    const pending = Promise.all([companies.loadCredentials(), companies.loadCompanies()])

    // Credentials resolve first; the slower companies load must keep it true.
    await Promise.resolve()
    await Promise.resolve()
    expect(companies.loading.value).toBe(true)

    resolveCompanies([])
    await pending
    expect(companies.loading.value).toBe(false)
  })
})

describe('companyOption', () => {
  it('labels a company with its name and formatted CNPJ', () => {
    const company = { CNPJ: '12345678000199', Name: 'Empresa Um' } as CompanySummary

    expect(companyOption(company)).toEqual({
      label: 'Empresa Um (12.345.678/0001-99)',
      value: '12345678000199',
    })
  })
})
