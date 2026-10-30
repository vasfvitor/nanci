import { desktopRuntime } from '@/platform/wails/runtime'
import type { Unsubscribe } from '@/platform/wails/events'
import { useConsoleStore } from '@/stores/console'

// NudgeResult says what nudgeWindow did: resized the window and put it back,
// left it alone because it is maximised, or skipped a nudge already running.
export type NudgeResult = 'resized' | 'maximised' | 'busy'

// snapshotDelayMs groups the focus and visibility events that arrive
// together into one window snapshot in the console.
export const snapshotDelayMs = 250

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

function errorText(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

// useDisplayScale works around WebView2 keeping the rasterisation of the old
// display scale (soft text) until a real relayout. start watches
// devicePixelRatio and, when it changes, widens the window by 1 px and puts
// it back; it also logs a snapshot of the scale and sizes at start, on focus
// and on visibility changes, only when a value changed. nudgeOnRequest does
// the same nudge when the user asks for it. See
// docs/pesquisa/2026-10-17-webview2-dpi.md.
export function useDisplayScale() {
  const consoleStore = useConsoleStore()
  let nudging = false

  function log(msg: string) {
    consoleStore.pushLogEntry({
      time: new Date().toISOString(),
      level: 'INFO',
      msg,
      attrs: '',
      raw: msg,
    })
  }

  // nudgeWindow waits a frame, so a resize Wails makes for the new scale has
  // landed, widens the window by 1 px and restores it on the next frame. A
  // maximised window is left alone: resizing it would restore it.
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
      log(`Janela reajustada a pedido (escala ${window.devicePixelRatio})`)
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

  function start(): Unsubscribe {
    let lastScale = window.devicePixelRatio
    let media: MediaQueryList | null = null
    let lastSnapshot = ''
    let snapshotTimer: ReturnType<typeof setTimeout> | undefined
    let stopped = false

    // A resolution query matches one ratio only, so it is replaced after
    // each change to listen for the next one.
    function watchScale() {
      media?.removeEventListener('change', checkScale)
      media = window.matchMedia(`(resolution: ${lastScale}dppx)`)
      media.addEventListener('change', checkScale)
    }

    function checkScale() {
      const scale = window.devicePixelRatio
      if (scale === lastScale) return
      const previous = lastScale
      lastScale = scale
      watchScale()
      void reactToScale(previous, scale)
    }

    async function reactToScale(previous: number, scale: number) {
      const change = `Escala do monitor mudou de ${previous} para ${scale}`
      try {
        const result = await nudgeWindow()
        if (result === 'resized') log(`${change}; janela reajustada`)
        if (result === 'maximised') log(`${change}; janela maximizada, sem reajuste`)
      } catch (error) {
        log(`${change}; falha ao reajustar a janela: ${errorText(error)}`)
      }
      scheduleSnapshot()
    }

    async function logSnapshot() {
      let windowSize = 'indisponível'
      try {
        const { w, h } = await desktopRuntime.windowGetSize()
        windowSize = `${w}x${h}`
      } catch {
        // Outside the Wails webview the runtime is missing; log the rest.
      }
      if (stopped) return
      const snapshot =
        `Janela: escala ${window.devicePixelRatio}, ` +
        `área ${window.innerWidth}x${window.innerHeight}, ` +
        `tela ${window.screen.width}x${window.screen.height}, ` +
        `janela ${windowSize}`
      if (snapshot === lastSnapshot) return
      lastSnapshot = snapshot
      log(snapshot)
    }

    function scheduleSnapshot() {
      if (stopped) return
      clearTimeout(snapshotTimer)
      snapshotTimer = setTimeout(() => void logSnapshot(), snapshotDelayMs)
    }

    function onWindowEvent() {
      checkScale()
      scheduleSnapshot()
    }

    watchScale()
    window.addEventListener('focus', onWindowEvent)
    document.addEventListener('visibilitychange', onWindowEvent)
    scheduleSnapshot()

    return () => {
      stopped = true
      clearTimeout(snapshotTimer)
      media?.removeEventListener('change', checkScale)
      media = null
      window.removeEventListener('focus', onWindowEvent)
      document.removeEventListener('visibilitychange', onWindowEvent)
    }
  }

  return { start, nudgeWindow, nudgeOnRequest, readDisplayInfo }
}
