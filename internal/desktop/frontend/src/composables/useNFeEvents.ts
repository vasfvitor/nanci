import { shallowRef } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { NFeEvent } from '@/types/desktop'

// useNFeEvents loads the events of one NF-e for a dialog. The list is
// disposable view state and resets on every load.
export function useNFeEvents() {
  const events = shallowRef<NFeEvent[]>([])
  const loading = shallowRef(false)

  async function load(cnpj: string, chaveAcesso: string) {
    events.value = []
    loading.value = true
    try {
      events.value = await desktopClient.listNFeEvents(cnpj, chaveAcesso)
    } finally {
      loading.value = false
    }
  }

  return { events, loading, load }
}
