import { shallowRef } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { CTeEvent } from '@/types/desktop'

// useCTeEvents loads the events of one CT-e for a dialog. The list is
// disposable view state and resets on every load.
export function useCTeEvents() {
  const events = shallowRef<CTeEvent[]>([])
  const loading = shallowRef(false)

  async function load(cnpj: string, chaveAcesso: string) {
    events.value = []
    loading.value = true
    try {
      events.value = await desktopClient.listCTeEvents(cnpj, chaveAcesso)
    } finally {
      loading.value = false
    }
  }

  return { events, loading, load }
}
