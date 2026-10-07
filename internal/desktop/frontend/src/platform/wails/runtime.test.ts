import { beforeEach, expect, vi } from 'vitest'
import { desktopRuntime } from './runtime'
import {
  Quit,
  WindowMinimise,
  WindowSetDarkTheme,
  WindowSetLightTheme,
  WindowSetSystemDefaultTheme,
  WindowToggleMaximise,
} from '../../../wailsjs/runtime/runtime'

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  Quit: vi.fn(),
  WindowMinimise: vi.fn(),
  WindowSetDarkTheme: vi.fn(),
  WindowSetLightTheme: vi.fn(),
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

  it('lets a runtime error reach the caller', () => {
    // Outside the Wails webview the runtime functions throw, as window.runtime
    // is missing; the wrapper does not hide that.
    vi.mocked(WindowSetDarkTheme).mockImplementation(() => {
      throw new TypeError("Cannot read properties of undefined (reading 'WindowSetDarkTheme')")
    })

    expect(() => desktopRuntime.setDarkTheme()).toThrow(TypeError)
  })
})
