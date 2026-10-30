import { ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { latestOnly } from '@/utils/latestOnly'

export type QueryType = 'nfse' | 'events'

// ChaveOption is a chave suggestion of the query form.
export type ChaveOption = { label: string; value: string; description?: string }

export const useQueryStore = defineStore('query', () => {
  // The company of a query is the workspace one; the form keeps the chave.
  const form = ref({
    chave: '',
  })
  const result = shallowRef('')
  const type = shallowRef<QueryType>('nfse')
  const loading = shallowRef(false)
  // chaveOptions are the chave suggestions listed for optionsCNPJ, or [] with
  // an empty optionsCNPJ while none are listed.
  const chaveOptions = shallowRef<ChaveOption[]>([])
  const optionsCNPJ = shallowRef('')
  // optionsGate lets only the latest suggestion fetch fill chaveOptions.
  const optionsGate = latestOnly()

  // controller aborts the query in flight. It lives here, next to loading, so
  // the query can be cancelled after leaving the page and coming back.
  let controller: AbortController | null = null

  // startQuery marks a query as running and returns the signal that cancels
  // it; finishQuery ends it, whatever its outcome.
  function startQuery(): AbortSignal {
    controller = new AbortController()
    loading.value = true
    return controller.signal
  }

  function finishQuery() {
    controller = null
    loading.value = false
  }

  function cancelQuery() {
    controller?.abort()
  }

  function clearResult() {
    result.value = ''
  }

  return {
    form,
    result,
    type,
    loading,
    chaveOptions,
    optionsCNPJ,
    optionsGate,
    startQuery,
    finishQuery,
    cancelQuery,
    clearResult,
  }
})
