import { computed, shallowRef, watch } from 'vue'
import type { useWorkspaceStore, WorkspaceKey } from '@/stores/workspace'

type ChaveFocus = { chave: string; cnpj: string; competence: string }

// chaveFocusState is the "show only this document" state the NF-e and CT-e
// stores share, set when the user follows a link from the other page. The
// focus belongs to the company and competência it was set under: once the
// workspace moves to another one, the focus is dropped and the page lists by
// the workspace again, also after the workspace comes back. onFocus runs
// when a focus is set, so the store can clear what would hide the document,
// such as the text filter.
export function chaveFocusState(
  workspace: ReturnType<typeof useWorkspaceStore>,
  onFocus: () => void = () => {}
) {
  const focus = shallowRef<ChaveFocus | null>(null)

  // The drop runs synchronously, before the page's workspace watchers search.
  watch(
    () => [workspace.cnpj, workspace.competence] as const,
    ([cnpj, competence]) => {
      const current = focus.value
      if (current && (current.cnpj !== cnpj || current.competence !== competence)) {
        focus.value = null
      }
    },
    { flush: 'sync' }
  )

  // chaveFocus is the access key the list is limited to, or ''.
  const chaveFocus = computed(() => focus.value?.chave ?? '')

  // rowsKey is the workspace key the listed rows belong to: a focused search
  // asks for every competência, but its rows still belong to the company and
  // competência of the workspace, so a remount on the same key keeps them.
  const rowsKey = computed<WorkspaceKey>(() => ({
    cnpj: workspace.cnpj,
    competence: workspace.competence,
  }))

  function focusChave(chave: string) {
    focus.value = { chave, cnpj: workspace.cnpj, competence: workspace.competence }
    onFocus()
  }

  function clearChaveFocus() {
    focus.value = null
  }

  return { chaveFocus, rowsKey, focusChave, clearChaveFocus }
}
