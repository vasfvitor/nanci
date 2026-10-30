import { Events } from '@wailsio/runtime'
import type { CertPasswordRequest, UpdateProgress } from '@/types/desktop'

export type Unsubscribe = () => void

// DesktopEvents is the payload callers receive for each event the backend
// registers (internal/desktop/events.go). The binding generator types the
// payloads in Events.CustomEvents; a payload with a hand-written frontend type
// uses that type instead.
export type DesktopEvents = Fits<
  Events.CustomEvents,
  {
    'backend-log': Events.CustomEvents['backend-log']
    'app:before-close': number
    'request-cert-password': CertPasswordRequest
    'notify-success': string
    'notify-error': string
    'wails:updater:download-progress': UpdateProgress
  }
>

export type DesktopEventName = keyof DesktopEvents

// Fits is Payloads when it names every registered event and each generated
// payload can be assigned to it. Otherwise a constraint fails, so a field the
// backend drops or retypes breaks the build here.
type Fits<
  Generated extends Payloads,
  Payloads extends Record<keyof Events.CustomEvents, unknown>,
> = Generated extends Payloads ? Payloads : never

// Wails v3 hands listeners a WailsEvent; the payload the backend emitted is
// its data field.
export function onWailsEvent<E extends DesktopEventName>(
  eventName: E,
  callback: (payload: DesktopEvents[E]) => void
): Unsubscribe
// Callers that name the payload type instead of the event still compile;
// the event name is checked, the payload type is trusted.
export function onWailsEvent<TPayload>(
  eventName: DesktopEventName,
  callback: (payload: TPayload) => void
): Unsubscribe
export function onWailsEvent(
  eventName: DesktopEventName,
  callback: (payload: unknown) => void
): Unsubscribe {
  const unsubscribe = Events.On(eventName, (event) => {
    callback(event.data)
  })

  return () => {
    unsubscribe()
  }
}
