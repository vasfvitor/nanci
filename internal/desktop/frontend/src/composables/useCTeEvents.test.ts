import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCTeEvents } from './useCTeEvents'
import { desktopClient } from '@/platform/wails/client'
import type { CTeEvent } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listCTeEvents: vi.fn() },
}))

describe('useCTeEvents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the events of one CT-e', async () => {
    const event = { ID: 'evt-1' } as CTeEvent
    vi.mocked(desktopClient.listCTeEvents).mockResolvedValue([event])

    const cteEvents = useCTeEvents()
    await cteEvents.load('123', 'chave-1')

    expect(desktopClient.listCTeEvents).toHaveBeenCalledWith('123', 'chave-1')
    expect(cteEvents.events.value).toEqual([event])
    expect(cteEvents.loading.value).toBe(false)
  })

  it('clears the list and the loading flag when the load fails', async () => {
    vi.mocked(desktopClient.listCTeEvents).mockResolvedValueOnce([{ ID: 'old' } as CTeEvent])
    const cteEvents = useCTeEvents()
    await cteEvents.load('123', 'chave-1')

    vi.mocked(desktopClient.listCTeEvents).mockRejectedValueOnce(new Error('boom'))
    await expect(cteEvents.load('123', 'chave-2')).rejects.toThrow('boom')

    expect(cteEvents.events.value).toEqual([])
    expect(cteEvents.loading.value).toBe(false)
  })
})
