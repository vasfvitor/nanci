import { computed, shallowRef, type Ref } from 'vue'
import { companyOption } from '@/composables/useCompanies'
import { desktopClient } from '@/platform/wails/client'
import type { CompanySummary } from '@/types/desktop'

// useCompanyFilter holds the company list of a document page and keeps
// cnpj, the company filter, pointing at a company that exists.
export function useCompanyFilter(cnpj: Ref<string>) {
  const companies = shallowRef<CompanySummary[]>([])
  const companyOptions = computed(() => companies.value.map(companyOption))
  const selectedCompany = computed(
    () => companies.value.find((company) => company.CNPJ === cnpj.value) ?? null
  )

  // loadCompanies lists the companies and picks the filter's company:
  // preferred (a CNPJ from the route) when it exists, else the current one
  // when it still exists, else the first company.
  async function loadCompanies(preferred = ''): Promise<CompanySummary[]> {
    const list = await desktopClient.listCompanies()
    companies.value = list
    const exists = (value: string) => Boolean(value) && list.some((item) => item.CNPJ === value)
    if (exists(preferred)) {
      cnpj.value = preferred
    } else if (!exists(cnpj.value)) {
      cnpj.value = list[0]?.CNPJ ?? ''
    }
    return list
  }

  return { companies, companyOptions, selectedCompany, loadCompanies }
}
