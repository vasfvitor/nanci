import {
  Quit,
  WindowMinimise,
  WindowSetDarkTheme,
  WindowSetLightTheme,
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
}
