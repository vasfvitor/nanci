import { computed, shallowRef } from 'vue'
import type { useWorkspaceStore } from '@/stores/workspace'

type ChaveFocus = { chave: string; cnpj: string; competence: string }

// chaveFocusState is the "show only this document" state the NF-e and CT-e
// stores share, set when the user follows a link from the other page. The
// focus belongs to the company and competência it was set under: once the
// workspace moves to another one, chaveFocus reads '' and the page lists by
// the workspace again.
export function chaveFocusState(workspace: ReturnType<typeof useWorkspaceStore>) {
  const focus = shallowRef<ChaveFocus | null>(null)

  // chaveFocus is the access key the list is limited to, or ''.
  const chaveFocus = computed(() => {
    const current = focus.value
    if (!current) return ''
    if (current.cnpj !== workspace.cnpj || current.competence !== workspace.competence) return ''
    return current.chave
  })

  function focusChave(chave: string) {
    focus.value = { chave, cnpj: workspace.cnpj, competence: workspace.competence }
  }

  function clearChaveFocus() {
    focus.value = null
  }

  return { chaveFocus, focusChave, clearChaveFocus }
}
