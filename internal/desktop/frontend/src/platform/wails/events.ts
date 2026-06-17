import { EventsOn } from '../../../wailsjs/runtime/runtime'

export type Unsubscribe = () => void

export function onWailsEvent<TPayload>(
  eventName: string,
  callback: (payload: TPayload) => void
): Unsubscribe {
  const unsubscribe = EventsOn(eventName, (payload: TPayload) => {
    callback(payload)
  })

  return () => {
    unsubscribe()
  }
}
