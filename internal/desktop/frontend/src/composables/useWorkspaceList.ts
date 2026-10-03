import { computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { sameWorkspaceKey, useWorkspaceStore, type WorkspaceKey } from '@/stores/workspace'

export type WorkspaceChange = {
  // company is true when the company changed since the rows were loaded, or
  // when the page has just mounted.
  company: boolean
}

export type WorkspaceListOptions = {
  // rowsFor returns the company and competência of the listed rows, or null.
  rowsFor: () => WorkspaceKey | null
  // clear drops the rows and the selection of another company or competência.
  clear: (change: WorkspaceChange) => void
  // reload searches the page list, and its status when the company changed.
  reload: (change: WorkspaceChange) => Promise<void>
}

// useWorkspaceList keeps a document page on the company and competência of
// the workspace. On mount and on every change it clears rows loaded for
// another key, so a page never shows them under the new header, then
// reloads when a company is selected. Remounting on the same key keeps the
// rows and the selection and refreshes them.
export function useWorkspaceList(options: WorkspaceListOptions) {
  const workspace = useWorkspaceStore()
  const { cnpj, loaded, loadError } = storeToRefs(workspace)

  // lastKey is the key this page last reloaded for, or null before its first
  // run. It stands in for rowsFor while a cleared list is still loading, so
  // stepping the competência twice does not count as a company change.
  let lastKey: WorkspaceKey | null = null
  watch(
    [cnpj, () => workspace.competence],
    ([companyCNPJ, competence]) => {
      const key = { cnpj: companyCNPJ, competence: competence || '' }
      const rowsFor = options.rowsFor()
      const company = (rowsFor ?? lastKey)?.cnpj !== companyCNPJ
      const mounting = lastKey === null
      lastKey = key
      if (!sameWorkspaceKey(rowsFor, key)) options.clear({ company })
      if (!companyCNPJ) return
      void options.reload({ company: company || mounting })
    },
    { immediate: true }
  )

  // noCompanyLabel is the empty-table text while no company is selected.
  const noCompanyLabel = computed(() => {
    if (loadError.value) return 'Não foi possível carregar as empresas.'
    if (!loaded.value) return 'Carregando empresas…'
    return 'Nenhuma empresa cadastrada. Cadastre uma em Empresas.'
  })

  return { cnpj, loaded, noCompanyLabel }
}
