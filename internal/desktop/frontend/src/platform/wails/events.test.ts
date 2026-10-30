import { beforeEach, expect, vi } from 'vitest'
import { Events } from '@wailsio/runtime'
import { onWailsEvent } from './events'

vi.mock('@wailsio/runtime', () => ({
  Events: { On: vi.fn() },
}))

describe('onWailsEvent', () => {
  beforeEach(() => {
    vi.mocked(Events.On).mockReset()
  })

  it('passes the event data to the callback and returns the Wails unsubscribe', () => {
    const unsubscribe = vi.fn()
    vi.mocked(Events.On).mockReturnValue(unsubscribe)

    const callback = vi.fn()
    const cleanup = onWailsEvent<string>('backend-log', callback)

    expect(Events.On).toHaveBeenCalledWith('backend-log', expect.any(Function))

    const firstCall = vi.mocked(Events.On).mock.calls[0]
    expect(firstCall).toBeDefined()
    if (!firstCall) return
    const [, handler] = firstCall
    handler({ name: 'backend-log', data: 'payload' } as Parameters<typeof handler>[0])
    expect(callback).toHaveBeenCalledWith('payload')

    cleanup()
    expect(unsubscribe).toHaveBeenCalledOnce()
  })
})
