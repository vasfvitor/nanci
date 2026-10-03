import { shallowRef } from 'vue'

// useEventList loads the events of one document for an events dialog. The
// list is disposable view state and empties at the start of every load, so
// a dialog never shows the events of the previous document.
export function useEventList<Event>(fetch: () => Promise<Event[]>) {
  const rows = shallowRef<Event[]>([])
  const loading = shallowRef(false)

  async function load() {
    rows.value = []
    loading.value = true
    try {
      rows.value = await fetch()
    } finally {
      loading.value = false
    }
  }

  return { rows, loading, load }
}
