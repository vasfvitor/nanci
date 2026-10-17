import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { snapshotDelayMs, useDisplayScale } from './useDisplayScale'
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

type ChangeListener = () => void

// fakeMedia stands in for matchMedia: it records each query and its change
// listeners, so a test can fire the query the composable listens to.
const fakeMedia = {
  queries: [] as string[],
  listeners: new Map<string, Set<ChangeListener>>(),
  matchMedia(query: string) {
    fakeMedia.queries.push(query)
    const listeners = fakeMedia.listeners.get(query) ?? new Set<ChangeListener>()
    fakeMedia.listeners.set(query, listeners)
    return {
      media: query,
      matches: true,
      addEventListener: (_type: string, listener: ChangeListener) => listeners.add(listener),
      removeEventListener: (_type: string, listener: ChangeListener) => listeners.delete(listener),
    } as unknown as MediaQueryList
  },
  fire(query: string) {
    for (const listener of [...(fakeMedia.listeners.get(query) ?? [])]) listener()
  },
  listenerCount() {
    let count = 0
    for (const listeners of fakeMedia.listeners.values()) count += listeners.size
    return count
  },
}

// settle drains the microtasks of a nudge; the timers stay fake.
async function settle() {
  for (let i = 0; i < 20; i++) await Promise.resolve()
}

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
    vi.useFakeTimers()
    fakeMedia.queries = []
    fakeMedia.listeners.clear()
    vi.stubGlobal('matchMedia', fakeMedia.matchMedia)
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      queueMicrotask(() => callback(0))
      return 0
    })
    vi.mocked(desktopRuntime.windowIsMaximised).mockResolvedValue(false)
    setScale(1)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('nudges the window and listens to the new ratio when the scale changes', async () => {
    const stop = useDisplayScale().start()
    expect(fakeMedia.queries).toEqual(['(resolution: 1dppx)'])

    setScale(1.25)
    fakeMedia.fire('(resolution: 1dppx)')
    await settle()

    expect(vi.mocked(desktopRuntime.windowSetSize).mock.calls).toEqual([
      [1281, 768],
      [1280, 768],
    ])
    expect(fakeMedia.queries).toEqual(['(resolution: 1dppx)', '(resolution: 1.25dppx)'])
    expect(fakeMedia.listeners.get('(resolution: 1dppx)')?.size).toBe(0)
    expect(logMessages()).toContain('Escala do monitor mudou de 1 para 1.25; janela reajustada')
    stop()
  })

  it('does not nudge when focus finds the same ratio', async () => {
    const stop = useDisplayScale().start()

    window.dispatchEvent(new Event('focus'))
    document.dispatchEvent(new Event('visibilitychange'))
    await settle()

    expect(desktopRuntime.windowSetSize).not.toHaveBeenCalled()
    stop()
  })

  it('checks the ratio again on focus', async () => {
    const stop = useDisplayScale().start()

    setScale(1.5)
    window.dispatchEvent(new Event('focus'))
    await settle()

    expect(desktopRuntime.windowSetSize).toHaveBeenCalledTimes(2)
    stop()
  })

  it('only logs when the window is maximised', async () => {
    vi.mocked(desktopRuntime.windowIsMaximised).mockResolvedValue(true)
    const stop = useDisplayScale().start()

    setScale(1.25)
    fakeMedia.fire('(resolution: 1dppx)')
    await settle()

    expect(desktopRuntime.windowSetSize).not.toHaveBeenCalled()
    expect(logMessages()).toContain(
      'Escala do monitor mudou de 1 para 1.25; janela maximizada, sem reajuste'
    )
    stop()
  })

  it('logs a window snapshot at start and again only when a value changed', async () => {
    const stop = useDisplayScale().start()
    await vi.advanceTimersByTimeAsync(snapshotDelayMs)
    const snapshots = () => logMessages().filter((msg) => msg.startsWith('Janela:'))

    expect(snapshots()).toEqual([
      `Janela: escala 1, área ${window.innerWidth}x${window.innerHeight}, ` +
        `tela ${window.screen.width}x${window.screen.height}, janela 1280x768`,
    ])

    window.dispatchEvent(new Event('focus'))
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(snapshotDelayMs)
    expect(snapshots()).toHaveLength(1)

    vi.mocked(desktopRuntime.windowGetSize).mockResolvedValue({ w: 1300, h: 768 })
    window.dispatchEvent(new Event('focus'))
    await vi.advanceTimersByTimeAsync(snapshotDelayMs)
    expect(snapshots()).toHaveLength(2)
    expect(snapshots()[1]).toContain('janela 1300x768')
    stop()
  })

  it('removes every listener on cleanup', async () => {
    const stop = useDisplayScale().start()
    expect(fakeMedia.listenerCount()).toBe(1)

    stop()
    setScale(2)
    window.dispatchEvent(new Event('focus'))
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(snapshotDelayMs)

    expect(fakeMedia.listenerCount()).toBe(0)
    expect(desktopRuntime.windowSetSize).not.toHaveBeenCalled()
    expect(logMessages()).toEqual([])
  })

  it('nudges on request and reports a maximised window', async () => {
    const displayScale = useDisplayScale()

    await expect(displayScale.nudgeOnRequest()).resolves.toBe('resized')
    expect(desktopRuntime.windowSetSize).toHaveBeenCalledTimes(2)
    expect(logMessages()).toEqual(['Janela reajustada a pedido (escala 1)'])

    vi.mocked(desktopRuntime.windowIsMaximised).mockResolvedValue(true)
    await expect(displayScale.nudgeOnRequest()).resolves.toBe('maximised')
    expect(desktopRuntime.windowSetSize).toHaveBeenCalledTimes(2)
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
