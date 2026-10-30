import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useDisplayScale } from './useDisplayScale'
import { desktopRuntime } from '@/platform/wails/runtime'
import { useConsoleStore } from '@/stores/console'

vi.mock('@/platform/wails/runtime', () => ({
  desktopRuntime: {
    windowGetSize: vi.fn(async () => ({ w: 1280, h: 768 })),
    windowSetSize: vi.fn(),
    windowIsMaximised: vi.fn(async () => false),
    screenGetAll: vi.fn(async () => [{}, {}]),
  },
}))

function setScale(scale: number) {
  Object.defineProperty(window, 'devicePixelRatio', { value: scale, configurable: true })
}

function logMessages() {
  return useConsoleStore().logEntries.map((entry) => entry.msg)
}

describe('useDisplayScale', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      queueMicrotask(() => callback(0))
      return 0
    })
    vi.mocked(desktopRuntime.windowIsMaximised).mockResolvedValue(false)
    setScale(1)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('widens the window by 1 px and puts it back on request', async () => {
    const displayScale = useDisplayScale()

    await expect(displayScale.nudgeOnRequest()).resolves.toBe('resized')
    expect(vi.mocked(desktopRuntime.windowSetSize).mock.calls).toEqual([
      [1281, 768],
      [1280, 768],
    ])
    expect(logMessages()).toEqual(['Janela reajustada a pedido (escala 1)'])
  })

  it('leaves a maximised window alone', async () => {
    vi.mocked(desktopRuntime.windowIsMaximised).mockResolvedValue(true)

    await expect(useDisplayScale().nudgeOnRequest()).resolves.toBe('maximised')
    expect(desktopRuntime.windowSetSize).not.toHaveBeenCalled()
    expect(logMessages()).toEqual([])
  })

  it('skips a second request while a nudge is running', async () => {
    const displayScale = useDisplayScale()

    const first = displayScale.nudgeOnRequest()
    await expect(displayScale.nudgeOnRequest()).resolves.toBe('busy')
    await expect(first).resolves.toBe('resized')
    expect(desktopRuntime.windowSetSize).toHaveBeenCalledTimes(2)
  })

  it('does not watch the display on its own', () => {
    const matchMedia = vi.fn()
    vi.stubGlobal('matchMedia', matchMedia)

    useDisplayScale()
    setScale(1.25)
    window.dispatchEvent(new Event('focus'))
    document.dispatchEvent(new Event('visibilitychange'))

    expect(matchMedia).not.toHaveBeenCalled()
    expect(desktopRuntime.windowIsMaximised).not.toHaveBeenCalled()
    expect(desktopRuntime.windowSetSize).not.toHaveBeenCalled()
  })

  it('reads the scale, screen size and monitor count', async () => {
    setScale(1.5)

    await expect(useDisplayScale().readDisplayInfo()).resolves.toEqual({
      scale: 1.5,
      screenSize: `${window.screen.width}x${window.screen.height}`,
      monitors: 2,
    })

    vi.mocked(desktopRuntime.screenGetAll).mockRejectedValue(new Error('no runtime'))
    await expect(useDisplayScale().readDisplayInfo()).resolves.toMatchObject({ monitors: null })
  })
})
