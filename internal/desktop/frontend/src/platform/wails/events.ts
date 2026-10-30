import { Events } from '@wailsio/runtime'

export type Unsubscribe = () => void

// Wails v3 hands listeners a WailsEvent; the payload the backend emitted is
// its data field.
export function onWailsEvent<TPayload>(
  eventName: string,
  callback: (payload: TPayload) => void
): Unsubscribe {
  const unsubscribe = Events.On(eventName, (event) => {
    callback(event.data as TPayload)
  })

  return () => {
    unsubscribe()
  }
}
