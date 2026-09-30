import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFSeEvents } from './useNFSeEvents'
import { desktopClient } from '@/platform/wails/client'
import type { DocumentEvent } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listEventsForDocument: vi.fn() },
}))

describe('useNFSeEvents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the events of one NFS-e', async () => {
    const event = { ID: 'evt-1' } as DocumentEvent
    vi.mocked(desktopClient.listEventsForDocument).mockResolvedValue([event])

    const nfseEvents = useNFSeEvents()
    await nfseEvents.load('doc-1')

    expect(desktopClient.listEventsForDocument).toHaveBeenCalledWith('doc-1')
    expect(nfseEvents.events.value).toEqual([event])
    expect(nfseEvents.loading.value).toBe(false)
  })

  it('clears the list and the loading flag when the load fails', async () => {
    vi.mocked(desktopClient.listEventsForDocument).mockResolvedValueOnce([
      { ID: 'old' } as DocumentEvent,
    ])
    const nfseEvents = useNFSeEvents()
    await nfseEvents.load('doc-1')

    vi.mocked(desktopClient.listEventsForDocument).mockRejectedValueOnce(new Error('boom'))
    await expect(nfseEvents.load('doc-2')).rejects.toThrow('boom')

    expect(nfseEvents.events.value).toEqual([])
    expect(nfseEvents.loading.value).toBe(false)
  })
})
