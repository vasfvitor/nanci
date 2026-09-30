import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCompanyFilter } from './useCompanyFilter'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listCompanies: vi.fn() },
}))

function company(cnpj: string, name: string) {
  return { CNPJ: cnpj, Name: name, Environment: 'producao' } as CompanySummary
}

const um = company('11111111000111', 'Empresa Um')
const dois = company('22222222000122', 'Empresa Dois')

describe('useCompanyFilter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([um, dois])
  })

  it('lists the companies as options and exposes the selected one', async () => {
    const cnpj = ref('')
    const filter = useCompanyFilter(cnpj)

    await filter.loadCompanies()

    expect(filter.companyOptions.value).toEqual([
      { label: 'Empresa Um (11111111000111)', value: '11111111000111' },
      { label: 'Empresa Dois (22222222000122)', value: '22222222000122' },
    ])
    expect(cnpj.value).toBe(um.CNPJ)
    expect(filter.selectedCompany.value).toEqual(um)
  })

  it('keeps a company that still exists', async () => {
    const cnpj = ref(dois.CNPJ)
    await useCompanyFilter(cnpj).loadCompanies()
    expect(cnpj.value).toBe(dois.CNPJ)
  })

  it('prefers the company from the route when it exists', async () => {
    const cnpj = ref(um.CNPJ)
    const filter = useCompanyFilter(cnpj)

    await filter.loadCompanies(dois.CNPJ)
    expect(cnpj.value).toBe(dois.CNPJ)

    await filter.loadCompanies('99999999000199')
    expect(cnpj.value).toBe(dois.CNPJ)
  })

  it('falls back to the first company when the selected one is gone', async () => {
    const cnpj = ref('99999999000199')
    await useCompanyFilter(cnpj).loadCompanies()
    expect(cnpj.value).toBe(um.CNPJ)
  })

  it('clears the filter when there are no companies', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    const cnpj = ref(um.CNPJ)
    const filter = useCompanyFilter(cnpj)

    await filter.loadCompanies()

    expect(cnpj.value).toBe('')
    expect(filter.selectedCompany.value).toBeNull()
  })
})
