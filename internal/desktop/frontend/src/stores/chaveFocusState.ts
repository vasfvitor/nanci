import { computed, shallowRef, watch, type Ref } from 'vue'
import type { useWorkspaceStore, WorkspaceKey } from '@/stores/workspace'

type ChaveFocus = { chave: string; cnpj: string; competence: string }

// chaveFocusState is the "show only this document" state the NF-e and CT-e
// stores share, set when the user follows a link from the other page. The
// focus belongs to the company and competência it was set under: under
// another one chaveFocus reads '', and the focus is dropped so it does not
// come back with the workspace. Focusing clears filterText, the text typed
// for the list, so it cannot hide the document.
export function chaveFocusState(
  workspace: ReturnType<typeof useWorkspaceStore>,
  filterText: Ref<string>
) {
  const focus = shallowRef<ChaveFocus | null>(null)
  const onFocusKey = (current: ChaveFocus) =>
    current.cnpj === workspace.cnpj && current.competence === workspace.competence

  // chaveFocus is the access key the list is limited to, or ''.
  const chaveFocus = computed(() => {
    const current = focus.value
    return current && onFocusKey(current) ? current.chave : ''
  })

  // Synchronous, so leaving the key and coming back within one tick still
  // drops the focus; chaveFocus does not depend on it.
  watch(
    () => [workspace.cnpj, workspace.competence],
    () => {
      if (focus.value && !onFocusKey(focus.value)) focus.value = null
    },
    { flush: 'sync' }
  )

  // rowsKey is the workspace key the listed rows belong to: a focused search
  // asks for every competência, but its rows still belong to the company and
  // competência of the workspace, so a remount on the same key keeps them.
  const rowsKey = computed<WorkspaceKey>(() => ({
    cnpj: workspace.cnpj,
    competence: workspace.competence,
  }))

  function focusChave(chave: string) {
    focus.value = { chave, cnpj: workspace.cnpj, competence: workspace.competence }
    filterText.value = ''
  }

  function clearChaveFocus() {
    focus.value = null
  }

  return { chaveFocus, rowsKey, focusChave, clearChaveFocus }
}
