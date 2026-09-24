import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeEvents } from './useNFeEvents'
import { desktopClient } from '@/platform/wails/client'
import type { NFeEvent } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listNFeEvents: vi.fn() },
}))

describe('useNFeEvents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the events of one note', async () => {
    const event = { ID: 'evt-1' } as NFeEvent
    vi.mocked(desktopClient.listNFeEvents).mockResolvedValue([event])

    const nfeEvents = useNFeEvents()
    await nfeEvents.load('123', 'chave-1')

    expect(desktopClient.listNFeEvents).toHaveBeenCalledWith('123', 'chave-1')
    expect(nfeEvents.events.value).toEqual([event])
    expect(nfeEvents.loading.value).toBe(false)
  })

  it('clears the list and the loading flag when the load fails', async () => {
    vi.mocked(desktopClient.listNFeEvents).mockResolvedValueOnce([{ ID: 'old' } as NFeEvent])
    const nfeEvents = useNFeEvents()
    await nfeEvents.load('123', 'chave-1')

    vi.mocked(desktopClient.listNFeEvents).mockRejectedValueOnce(new Error('boom'))
    await expect(nfeEvents.load('123', 'chave-2')).rejects.toThrow('boom')

    expect(nfeEvents.events.value).toEqual([])
    expect(nfeEvents.loading.value).toBe(false)
  })
})
