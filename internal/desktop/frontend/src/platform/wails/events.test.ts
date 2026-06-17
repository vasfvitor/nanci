import { beforeEach, expect, vi } from 'vitest'
import { onWailsEvent } from './events'
import { EventsOn } from '../../../wailsjs/runtime/runtime'

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
}))

describe('onWailsEvent', () => {
  beforeEach(() => {
    vi.mocked(EventsOn).mockReset()
  })

  it('returns the unsubscribe function provided by Wails', () => {
    const unsubscribe = vi.fn()
    vi.mocked(EventsOn).mockReturnValue(unsubscribe)

    const callback = vi.fn()
    const cleanup = onWailsEvent<string>('backend-log', callback)

    expect(EventsOn).toHaveBeenCalledWith('backend-log', expect.any(Function))

    const firstCall = vi.mocked(EventsOn).mock.calls[0]
    expect(firstCall).toBeDefined()
    if (!firstCall) return
    const [, handler] = firstCall
    handler('payload')
    expect(callback).toHaveBeenCalledWith('payload')

    cleanup()
    expect(unsubscribe).toHaveBeenCalledOnce()
  })
})
