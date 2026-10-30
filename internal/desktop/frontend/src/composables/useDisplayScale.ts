import { desktopRuntime } from '@/platform/wails/runtime'
import { useConsoleStore } from '@/stores/console'

// NudgeResult says what nudgeOnRequest did: resized the window and put it
// back, left it alone because it is maximised, or skipped a nudge already
// running.
export type NudgeResult = 'resized' | 'maximised' | 'busy'

// DisplayInfo is what the Sobre section shows about the display: the scale
// of the monitor the window is on, its size, and how many monitors Wails
// lists (null when the runtime cannot tell).
export type DisplayInfo = {
  scale: number
  screenSize: string
  monitors: number | null
}

function nextFrame(): Promise<void> {
  return new Promise((resolve) => requestAnimationFrame(() => resolve()))
}

// useDisplayScale backs the display rows and the "Reajustar janela" button in
// Configurações › Sobre. Wails follows display scale changes itself; the
// button stays as a diagnostic for a window that looks soft. See
// docs/pesquisa/2026-10-17-webview2-dpi.md.
export function useDisplayScale() {
  const consoleStore = useConsoleStore()
  let nudging = false

  // nudgeWindow waits a frame, widens the window by 1 px and restores it on
  // the next frame, which forces a relayout of the webview. A maximised
  // window is left alone: resizing it would restore it.
  async function nudgeWindow(): Promise<NudgeResult> {
    if (nudging) return 'busy'
    nudging = true
    try {
      if (await desktopRuntime.windowIsMaximised()) return 'maximised'
      await nextFrame()
      const { w, h } = await desktopRuntime.windowGetSize()
      await desktopRuntime.windowSetSize(w + 1, h)
      await nextFrame()
      await desktopRuntime.windowSetSize(w, h)
      return 'resized'
    } finally {
      nudging = false
    }
  }

  async function nudgeOnRequest(): Promise<NudgeResult> {
    const result = await nudgeWindow()
    if (result === 'resized') {
      const msg = `Janela reajustada a pedido (escala ${window.devicePixelRatio})`
      consoleStore.pushLogEntry({
        time: new Date().toISOString(),
        level: 'INFO',
        msg,
        attrs: '',
        raw: msg,
      })
    }
    return result
  }

  async function readDisplayInfo(): Promise<DisplayInfo> {
    let monitors: number | null = null
    try {
      monitors = (await desktopRuntime.screenGetAll()).length
    } catch {
      // Outside the Wails webview the runtime is missing.
    }
    return {
      scale: window.devicePixelRatio,
      screenSize: `${window.screen.width}x${window.screen.height}`,
      monitors,
    }
  }

  return { nudgeOnRequest, readDisplayInfo }
}
