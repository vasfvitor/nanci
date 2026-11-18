import { shallowRef, watch } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { CTeRow } from '@/types/desktop'
import { latestOnly } from '@/utils/latestOnly'

// useNFeLinkedCTe loads the CT-e of the company that transported one NF-e,
// from every competência, for the expanded row of that note. The result
// lives only while the row is open, so it stays here instead of a store.
export function useNFeLinkedCTe(cnpj: () => string, chave: () => string) {
  const rows = shallowRef<CTeRow[]>([])
  const loading = shallowRef(false)
  const error = shallowRef('')
  // gate lets only the load of the current company and note fill rows.
  const gate = latestOnly()

  async function load(companyCNPJ: string, nfeChave: string) {
    const isLatest = gate.begin()
    rows.value = []
    error.value = ''
    if (!companyCNPJ || !nfeChave) return
    loading.value = true
    try {
      const result = await desktopClient.listCTe({
        CNPJ: companyCNPJ,
        Competence: '',
        Situacao: '',
        Role: '',
        Modelo: '',
        EmitenteCNPJ: '',
        TomadorCNPJ: '',
        NFeChave: nfeChave,
        OnlyUnread: false,
      })
      if (isLatest()) rows.value = result
    } catch {
      if (isLatest()) error.value = 'Não foi possível carregar os CT-e desta NF-e.'
    } finally {
      if (isLatest()) loading.value = false
    }
  }

  watch([cnpj, chave], ([companyCNPJ, nfeChave]) => void load(companyCNPJ, nfeChave), {
    immediate: true,
  })

  return { rows, loading, error }
}
