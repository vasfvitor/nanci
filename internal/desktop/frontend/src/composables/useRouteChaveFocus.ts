import { watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { compactCode } from '@/utils/cnpj'
import { isChaveDFe } from '@/utils/formatters'

// useRouteChaveFocus reads "?chave=", which the links between the NF-e and
// CT-e pages set, focuses the page list on that document and drops the
// query, so going back to the page does not focus it again. Call it before
// useWorkspaceList: the first read runs during setup, so the page's first
// search is already focused. A later "?chave=" on the mounted page searches
// at once.
export function useRouteChaveFocus(focusChave: (chave: string) => void, search: () => unknown) {
  const route = useRoute()
  const router = useRouter()
  let setUp = false

  watch(
    () => route.query['chave'],
    (raw) => {
      if (!raw) return
      void router.replace({ query: {} })
      const chave = typeof raw === 'string' ? compactCode(raw) : ''
      if (!isChaveDFe(chave)) return
      focusChave(chave)
      if (setUp) void search()
    },
    { immediate: true }
  )
  setUp = true
}
