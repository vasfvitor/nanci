import { describe, expect, it, vi } from 'vitest'
import { useEventList } from './useEventList'

describe('useEventList', () => {
  it('loads the events of one document', async () => {
    const fetch = vi.fn(async () => [{ ID: 'evt-1' }])
    const events = useEventList(fetch)

    await events.load()

    expect(fetch).toHaveBeenCalledTimes(1)
    expect(events.rows.value).toEqual([{ ID: 'evt-1' }])
    expect(events.loading.value).toBe(false)
  })

  it('clears the list and the loading flag when the load fails', async () => {
    const fetch = vi.fn<() => Promise<{ ID: string }[]>>().mockResolvedValueOnce([{ ID: 'old' }])
    const events = useEventList(fetch)
    await events.load()

    fetch.mockRejectedValueOnce(new Error('boom'))
    await expect(events.load()).rejects.toThrow('boom')

    expect(events.rows.value).toEqual([])
    expect(events.loading.value).toBe(false)
  })
})
