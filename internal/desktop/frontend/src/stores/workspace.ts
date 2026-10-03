import { computed, ref, shallowRef, watch } from 'vue'
import { defineStore } from 'pinia'
import { desktopClient, errorMessage } from '@/platform/wails/client'
import type { CompanySummary } from '@/types/desktop'
import { isCompetence } from '@/utils/competence'

// WorkspaceKey is the company and competência a document list was loaded for.
export type WorkspaceKey = { cnpj: string; competence: string }

// sameWorkspaceKey reports whether a, which may be missing, is the key b.
export function sameWorkspaceKey(a: WorkspaceKey | null, b: WorkspaceKey): boolean {
  return a !== null && a.cnpj === b.cnpj && a.competence === b.competence
}

const cnpjKey = 'nanci:workspace:cnpj'
const competenceKey = 'nanci:workspace:competence'

// The store holds the company and competência every document page shows,
// and the company list they are picked from. Both choices survive restarts.
export const useWorkspaceStore = defineStore('workspace', () => {
  // storedCNPJ is the last company chosen, kept until a company list says
  // whether it still exists. cnpj only takes it from loadCompanies, so cnpj
  // never names a company that is not listed.
  let storedCNPJ = localStorage.getItem(cnpjKey) ?? ''
  const savedCompetence = localStorage.getItem(competenceKey) ?? ''

  // cnpj is a listed company, or '' while the list is not loaded or empty.
  const cnpj = ref('')
  // competence is a YYYY-MM competência, or '' for every month.
  const competence = ref(isCompetence(savedCompetence) ? savedCompetence : '')
  const companies = shallowRef<CompanySummary[]>([])
  // loaded is true once a company list arrived.
  const loaded = shallowRef(false)
  // loadError is the message of the failed latest load, or ''.
  const loadError = shallowRef('')

  const selectedCompany = computed(
    () => companies.value.find((company) => company.CNPJ === cnpj.value) ?? null
  )

  // An empty cnpj only means no company is listed, so it is never saved and
  // the stored choice comes back once the company exists again.
  watch(cnpj, (value) => {
    if (!value) return
    storedCNPJ = value
    localStorage.setItem(cnpjKey, value)
  })

  watch(competence, (value) => {
    localStorage.setItem(competenceKey, value)
  })

  // loadSeq numbers the loads; only the latest one fills the store.
  let loadSeq = 0
  // inFlight is the latest load while it runs.
  let inFlight: Promise<CompanySummary[]> | null = null

  async function fetchCompanies(seq: number): Promise<CompanySummary[]> {
    try {
      const list = await desktopClient.listCompanies()
      if (seq === loadSeq) {
        const exists = (value: string) => Boolean(value) && list.some((item) => item.CNPJ === value)
        // Set together, so a watcher of any of them runs once.
        companies.value = list
        cnpj.value = [cnpj.value, storedCNPJ].find(exists) ?? list[0]?.CNPJ ?? ''
        loaded.value = true
        loadError.value = ''
      }
      return list
    } catch (error) {
      if (seq === loadSeq) loadError.value = errorMessage(error)
      throw error
    } finally {
      if (seq === loadSeq) inFlight = null
    }
  }

  // loadCompanies always lists the companies and keeps cnpj on a listed
  // company: the current one, else the stored one, else the first. A load
  // that ends after a newer one started leaves the store as it is.
  function loadCompanies(): Promise<CompanySummary[]> {
    const request = fetchCompanies(++loadSeq)
    inFlight = request
    return request
  }

  // ensureCompanies returns the list of the load in flight, else the list
  // already loaded, and only lists the companies when neither exists.
  function ensureCompanies(): Promise<CompanySummary[]> {
    if (inFlight) return inFlight
    if (loaded.value) return Promise.resolve(companies.value)
    return loadCompanies()
  }

  return {
    cnpj,
    competence,
    companies,
    loaded,
    loadError,
    selectedCompany,
    loadCompanies,
    ensureCompanies,
  }
})
