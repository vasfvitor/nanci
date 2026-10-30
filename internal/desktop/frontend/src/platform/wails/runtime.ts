import { Application, Screens, Window } from '@wailsio/runtime'
import { SetWindowTheme } from '../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'

// The window controls are fire-and-forget: a failed call has nothing for the
// UI to recover.
function ignoreResult(call: () => Promise<unknown>): () => void {
  return () => {
    call().catch(() => undefined)
  }
}

export const desktopRuntime = {
  minimiseWindow: ignoreResult(() => Window.Minimise()),
  toggleMaximiseWindow: ignoreResult(() => Window.ToggleMaximise()),
  quit: ignoreResult(() => Application.Quit()),
  // Takes Quasar's dark setting. The Wails v3 JS runtime has no window theme
  // calls, so the theme goes through the App.SetWindowTheme binding.
  setWindowTheme(dark: boolean | 'auto'): void {
    ignoreResult(() => SetWindowTheme(dark === 'auto' ? 'system' : dark ? 'dark' : 'light'))()
  },
  // Sizes are in logical pixels: Wails scales them by the window DPI.
  async windowGetSize(): Promise<{ w: number; h: number }> {
    const { width, height } = await Window.Size()
    return { w: width, h: height }
  },
  // Every v3 runtime call is its own HTTP request, so a caller that resizes
  // twice awaits the first call to keep the order.
  windowSetSize: (width: number, height: number) => Window.SetSize(width, height),
  windowIsMaximised: () => Window.IsMaximised(),
  screenGetAll: () => Screens.GetAll(),
}
