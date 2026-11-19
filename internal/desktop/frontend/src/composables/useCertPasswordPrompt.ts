import { computed, onMounted, onUnmounted, ref } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import { onWailsEvent, type Unsubscribe } from '@/platform/wails/events'
import type { CertPasswordRequest } from '@/types/desktop'

// useCertPasswordPrompt queues the certificate password requests the backend
// sends ("request-cert-password") while the calling component is mounted, and
// answers them one at a time, oldest first. Call it from a component's setup.
export function useCertPasswordPrompt() {
  const requests = ref<CertPasswordRequest[]>([])
  // submitting is set while the answer to the current request is sent, so a
  // second click cannot answer it twice.
  const submitting = ref(false)
  const current = computed(() => requests.value[0])
  let unsubscribe: Unsubscribe | null = null

  // answer sends the answer to the current request and moves to the next one.
  // It reports whether it answered.
  async function answer(send: (req: CertPasswordRequest) => Promise<void>) {
    const req = requests.value[0]
    if (!req || submitting.value) return false
    submitting.value = true
    try {
      await send(req)
      requests.value.shift()
      return true
    } finally {
      submitting.value = false
    }
  }

  function submit(password: string) {
    return answer((req) => desktopClient.submitCertPassword(req.RequestID, password))
  }

  function cancel() {
    return answer((req) => desktopClient.cancelCertPassword(req.RequestID))
  }

  onMounted(() => {
    unsubscribe = onWailsEvent('request-cert-password', (req) => {
      requests.value.push(req)
    })
  })

  onUnmounted(() => {
    unsubscribe?.()
    unsubscribe = null
  })

  return { current, submitting, submit, cancel }
}
