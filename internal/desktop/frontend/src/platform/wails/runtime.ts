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

export const desktopRuntime = {
  minimiseWindow: WindowMinimise,
  toggleMaximiseWindow: WindowToggleMaximise,
  quit: Quit,
  setDarkTheme: WindowSetDarkTheme,
  setLightTheme: WindowSetLightTheme,
  setSystemDefaultTheme: WindowSetSystemDefaultTheme,
  // Sizes are in logical pixels: Wails scales them by the window DPI.
  windowGetSize: WindowGetSize,
  windowSetSize: WindowSetSize,
  windowIsMaximised: WindowIsMaximised,
  screenGetAll: ScreenGetAll,
}
