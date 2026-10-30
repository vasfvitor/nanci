import { beforeEach, expect, vi } from 'vitest'
import { Application, Screens, Window } from '@wailsio/runtime'
import { desktopRuntime } from './runtime'
import { SetWindowTheme } from '../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'

vi.mock('@wailsio/runtime', () => ({
  Application: { Quit: vi.fn(async () => undefined) },
  Screens: {
    GetAll: vi.fn(async () => [{ ID: '1', IsPrimary: true, Size: { Width: 1920, Height: 1080 } }]),
  },
  Window: {
    Minimise: vi.fn(async () => undefined),
    ToggleMaximise: vi.fn(async () => undefined),
    Size: vi.fn(async () => ({ width: 1280, height: 768 })),
    SetSize: vi.fn(async () => undefined),
    IsMaximised: vi.fn(async () => false),
  },
}))

vi.mock('../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app', () => ({
  SetWindowTheme: vi.fn(async () => undefined),
}))

describe('desktopRuntime', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each([
    ['minimiseWindow', Window.Minimise],
    ['toggleMaximiseWindow', Window.ToggleMaximise],
    ['quit', Application.Quit],
  ] as const)('%s calls its Wails runtime function', (name, call) => {
    desktopRuntime[name]()

    expect(call).toHaveBeenCalledOnce()
  })

  it.each([
    [true, 'dark'],
    [false, 'light'],
    ['auto', 'system'],
  ] as const)('setWindowTheme(%s) sends %s to App.SetWindowTheme', (dark, theme) => {
    desktopRuntime.setWindowTheme(dark)

    expect(SetWindowTheme).toHaveBeenCalledWith(theme)
  })

  it('passes the window size calls through, as {w, h}', async () => {
    await expect(desktopRuntime.windowGetSize()).resolves.toEqual({ w: 1280, h: 768 })
    await expect(desktopRuntime.windowIsMaximised()).resolves.toBe(false)
    await desktopRuntime.windowSetSize(1281, 768)

    expect(Window.Size).toHaveBeenCalledOnce()
    expect(Window.IsMaximised).toHaveBeenCalledOnce()
    expect(Window.SetSize).toHaveBeenCalledWith(1281, 768)
  })

  it('lists the screens', async () => {
    await expect(desktopRuntime.screenGetAll()).resolves.toHaveLength(1)
    expect(Screens.GetAll).toHaveBeenCalledOnce()
  })

  it('swallows a failed fire-and-forget call', async () => {
    // Outside the Wails webview the runtime call rejects; the window controls
    // have nothing to recover, so the rejection must not go unhandled.
    vi.mocked(Window.Minimise).mockRejectedValueOnce(new Error('Failed to fetch'))

    expect(() => desktopRuntime.minimiseWindow()).not.toThrow()
    await Promise.resolve()
  })

  it('lets a failed size query reach the caller', async () => {
    vi.mocked(Window.Size).mockRejectedValueOnce(new Error('Failed to fetch'))

    await expect(desktopRuntime.windowGetSize()).rejects.toThrow('Failed to fetch')
  })
})
