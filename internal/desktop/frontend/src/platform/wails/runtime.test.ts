import { beforeEach, expect, vi } from 'vitest'
import { desktopRuntime } from './runtime'
import {
  Quit,
  ScreenGetAll,
  WindowGetSize,
  WindowIsMaximised,
  WindowMinimise,
  WindowSetDarkTheme,
  WindowSetLightTheme,
  WindowSetSize,
  WindowSetSystemDefaultTheme,
  WindowToggleMaximise,
} from '../../../wailsjs/runtime/runtime'

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  Quit: vi.fn(),
  ScreenGetAll: vi.fn(async () => [
    { isCurrent: true, isPrimary: true, width: 1920, height: 1080 },
  ]),
  WindowGetSize: vi.fn(async () => ({ w: 1280, h: 768 })),
  WindowIsMaximised: vi.fn(async () => false),
  WindowMinimise: vi.fn(),
  WindowSetDarkTheme: vi.fn(),
  WindowSetLightTheme: vi.fn(),
  WindowSetSize: vi.fn(),
  WindowSetSystemDefaultTheme: vi.fn(),
  WindowToggleMaximise: vi.fn(),
}))

describe('desktopRuntime', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each([
    ['minimiseWindow', WindowMinimise],
    ['toggleMaximiseWindow', WindowToggleMaximise],
    ['quit', Quit],
    ['setDarkTheme', WindowSetDarkTheme],
    ['setLightTheme', WindowSetLightTheme],
    ['setSystemDefaultTheme', WindowSetSystemDefaultTheme],
  ] as const)('%s calls its Wails runtime function', (name, binding) => {
    desktopRuntime[name]()

    expect(binding).toHaveBeenCalledOnce()
  })

  it('passes the window size calls through', async () => {
    await expect(desktopRuntime.windowGetSize()).resolves.toEqual({ w: 1280, h: 768 })
    await expect(desktopRuntime.windowIsMaximised()).resolves.toBe(false)
    desktopRuntime.windowSetSize(1281, 768)

    expect(WindowGetSize).toHaveBeenCalledOnce()
    expect(WindowIsMaximised).toHaveBeenCalledOnce()
    expect(WindowSetSize).toHaveBeenCalledWith(1281, 768)
  })

  it('lists the screens', async () => {
    await expect(desktopRuntime.screenGetAll()).resolves.toHaveLength(1)
    expect(ScreenGetAll).toHaveBeenCalledOnce()
  })

  it('lets a runtime error reach the caller', () => {
    // Outside the Wails webview the runtime functions throw, as window.runtime
    // is missing; the wrapper does not hide that.
    vi.mocked(WindowSetDarkTheme).mockImplementation(() => {
      throw new TypeError("Cannot read properties of undefined (reading 'WindowSetDarkTheme')")
    })

    expect(() => desktopRuntime.setDarkTheme()).toThrow(TypeError)
  })
})
