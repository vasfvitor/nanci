import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { useCertPasswordPrompt } from './useCertPasswordPrompt'
import { desktopClient } from '@/platform/wails/client'
import type { CertPasswordRequest } from '@/types/desktop'

type Handler = (payload: unknown) => void

const handlers: Record<string, Handler> = {}
const unsubscribe = vi.fn()

vi.mock('@/platform/wails/events', () => ({
  onWailsEvent: (eventName: string, callback: Handler) => {
    handlers[eventName] = callback
    return unsubscribe
  },
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    submitCertPassword: vi.fn(),
    cancelCertPassword: vi.fn(),
  },
}))

function request(id: string): CertPasswordRequest {
  return { RequestID: id, CompanyName: `Empresa ${id}` } as CertPasswordRequest
}

// host mounts the composable in a component, as the dialog does.
function host() {
  let prompt!: ReturnType<typeof useCertPasswordPrompt>
  const wrapper = mount(
    defineComponent({
      setup() {
        prompt = useCertPasswordPrompt()
        return () => null
      },
    })
  )
  return { wrapper, prompt }
}

describe('useCertPasswordPrompt', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(desktopClient.submitCertPassword).mockResolvedValue(undefined)
    vi.mocked(desktopClient.cancelCertPassword).mockResolvedValue(undefined)
  })

  it('answers the requests oldest first', async () => {
    const { prompt } = host()
    handlers['request-cert-password']?.(request('a'))
    handlers['request-cert-password']?.(request('b'))
    expect(prompt.current.value?.RequestID).toBe('a')

    await expect(prompt.submit('segredo')).resolves.toBe(true)
    expect(desktopClient.submitCertPassword).toHaveBeenCalledWith('a', 'segredo')
    expect(prompt.current.value?.RequestID).toBe('b')

    await expect(prompt.cancel()).resolves.toBe(true)
    expect(desktopClient.cancelCertPassword).toHaveBeenCalledWith('b')
    expect(prompt.current.value).toBeUndefined()
  })

  it('keeps the request when the answer fails, and answers nothing without one', async () => {
    const { prompt } = host()
    await expect(prompt.submit('x')).resolves.toBe(false)
    expect(desktopClient.submitCertPassword).not.toHaveBeenCalled()

    handlers['request-cert-password']?.(request('a'))
    vi.mocked(desktopClient.submitCertPassword).mockRejectedValueOnce(new Error('senha errada'))
    await expect(prompt.submit('x')).rejects.toThrow('senha errada')
    await flushPromises()

    expect(prompt.current.value?.RequestID).toBe('a')
    expect(prompt.submitting.value).toBe(false)
  })

  it('stops listening when the component unmounts', () => {
    const { wrapper } = host()
    wrapper.unmount()
    expect(unsubscribe).toHaveBeenCalledOnce()
  })
})
