import { mount } from '@vue/test-utils'
import { defineComponent, onMounted, onUnmounted } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCloseGuard } from './useCloseGuard'
import { desktopClient } from '@/platform/wails/client'

type Handler = (payload: unknown) => void

const handlers: Record<string, Handler> = {}
const unsubscribe = vi.fn()
let okHandler: (() => void) | undefined
let dismissHandler: (() => void) | undefined

const dialog = vi.fn((_options: Record<string, unknown>) => {
  const chain = {
    onOk: (handler: () => void) => {
      okHandler = handler
      return chain
    },
    onDismiss: (handler: () => void) => {
      dismissHandler = handler
      return chain
    },
  }
  return chain
})

vi.mock('quasar', () => ({
  useQuasar: () => ({ dialog, notify: vi.fn() }),
  copyToClipboard: vi.fn(),
}))

vi.mock('@/platform/wails/events', () => ({
  onWailsEvent: (eventName: string, callback: Handler) => {
    handlers[eventName] = callback
    return unsubscribe
  },
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { confirmQuit: vi.fn(async () => undefined) },
  errorMessage: (e: unknown) => String(e),
  wailsErrorCode: () => '',
}))

// Host starts the guard the way MainLayout does.
const Host = defineComponent({
  setup() {
    const guard = useCloseGuard()
    let stop: (() => void) | undefined
    onMounted(() => {
      stop = guard.start()
    })
    onUnmounted(() => stop?.())
    return () => null
  },
})

function closeRequested() {
  handlers['app:before-close']?.(1)
}

describe('useCloseGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    okHandler = undefined
    dismissHandler = undefined
  })

  it('asks before closing and quits on Sair', () => {
    mount(Host)
    closeRequested()

    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Sincronização em andamento',
        ok: expect.objectContaining({ label: 'Sair' }),
        cancel: expect.objectContaining({ label: 'Continuar no Nanci' }),
        persistent: true,
      })
    )
    okHandler?.()
    expect(desktopClient.confirmQuit).toHaveBeenCalledTimes(1)
  })

  it('stays open on Continuar no Nanci', () => {
    mount(Host)
    closeRequested()
    dismissHandler?.()

    expect(desktopClient.confirmQuit).not.toHaveBeenCalled()
    closeRequested()
    expect(dialog).toHaveBeenCalledTimes(2)
  })

  it('does not stack prompts', () => {
    mount(Host)
    closeRequested()
    closeRequested()

    expect(dialog).toHaveBeenCalledTimes(1)
  })

  it('unsubscribes on unmount', () => {
    const wrapper = mount(Host)
    expect(unsubscribe).not.toHaveBeenCalled()

    wrapper.unmount()
    expect(unsubscribe).toHaveBeenCalledTimes(1)
  })
})
