import { computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useWorkspaceStore, type WorkspaceKey } from '@/stores/workspace'
import { workspaceContextLine } from '@/utils/formatters'

export type WorkspaceListOptions = {
  // rowsFor returns the company and competência of the listed rows, or null.
  rowsFor: () => WorkspaceKey | null
  // clearRows drops the rows and the selection of another key.
  clearRows: () => void
  // loadRows searches the page list.
  loadRows: () => Promise<void>
  // clearCompany drops company state, such as a sync status, that is not of
  // cnpj. A page without such state leaves it and loadCompany out.
  clearCompany?: (cnpj: string) => void
  // loadCompany loads the company state of the workspace company.
  loadCompany?: () => Promise<void>
}

// useWorkspaceList keeps a document page on the company and competência of
// the workspace. On mount and on every change it clears what was loaded for
// another company or key, so a page never shows it under the new header,
// then reloads while a company is selected: the company state when the
// company changes, the rows when the company or the competência does.
// Remounting on the same key keeps the rows and the selection and
// refreshes them.
export function useWorkspaceList(options: WorkspaceListOptions) {
  const workspace = useWorkspaceStore()
  const { cnpj, loaded, loadError } = storeToRefs(workspace)

  watch(
    cnpj,
    (companyCNPJ) => {
      options.clearCompany?.(companyCNPJ)
      if (companyCNPJ) void options.loadCompany?.()
    },
    { immediate: true }
  )

  watch(
    [cnpj, () => workspace.competence],
    ([companyCNPJ, competence]) => {
      const rowsFor = options.rowsFor()
      if (rowsFor?.cnpj !== companyCNPJ || rowsFor.competence !== competence) options.clearRows()
      if (companyCNPJ) void options.loadRows()
    },
    { immediate: true }
  )

  // noCompanyLabel is the empty-table text while no company is selected.
  const noCompanyLabel = computed(() => {
    if (loadError.value) return 'Não foi possível carregar as empresas.'
    if (!loaded.value) return 'Carregando empresas…'
    return 'Nenhuma empresa cadastrada. Cadastre uma em Empresas.'
  })

  // contextLine names the company and competência under the page title.
  const contextLine = computed(() =>
    workspaceContextLine(workspace.selectedCompany, workspace.competence)
  )

  return { cnpj, noCompanyLabel, contextLine }
}
