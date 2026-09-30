import { shallowRef } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import type { DocumentEvent } from '@/types/desktop'

// useNFSeEvents loads the events of one NFS-e for a dialog. The list is
// disposable view state and resets on every load.
export function useNFSeEvents() {
  const events = shallowRef<DocumentEvent[]>([])
  const loading = shallowRef(false)

  async function load(documentID: string) {
    events.value = []
    loading.value = true
    try {
      events.value = await desktopClient.listEventsForDocument(documentID)
    } finally {
      loading.value = false
    }
  }

  return { events, loading, load }
}
