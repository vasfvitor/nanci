import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useSourceSync } from './useSourceSync'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { cancelSync: vi.fn() },
}))

describe('useSourceSync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('follows the companySync markers of the company and source only', () => {
    const cnpj = ref('123')
    const resetting = ref('')
    const sync = useSourceSync('nfe', cnpj, resetting)
    const store = useCompanySyncStore()

    expect(sync.isSyncing.value).toBe(false)
    expect(sync.syncBusy('123')).toBe(false)
    expect(sync.syncBusy('')).toBe(true)

    store.startSync('123', 'cte')
    expect(sync.isSyncing.value).toBe(false)

    store.startSync('123', 'nfe')
    expect(sync.isSyncing.value).toBe(true)
    expect(sync.syncBusy('123')).toBe(true)
    expect(sync.syncBusy('456')).toBe(false)

    cnpj.value = '456'
    expect(sync.isSyncing.value).toBe(false)
  })

  it('counts the reset of the company as busy', () => {
    const resetting = ref('123')
    const sync = useSourceSync('cte', ref('123'), resetting)

    expect(sync.isResetting.value).toBe(true)
    expect(sync.syncBusy('123')).toBe(true)
    resetting.value = ''
    expect(sync.isResetting.value).toBe(false)
  })

  it('cancels the cancellable sync of the company and source', async () => {
    vi.mocked(desktopClient.cancelSync).mockResolvedValue(undefined)
    const sync = useSourceSync('nfe', ref('123'), ref(''))
    const store = useCompanySyncStore()
    const running = store.runCancellableSync('123', 'nfe', () => new Promise(() => {}))
    void running

    expect(sync.canCancelSync.value).toBe(true)
    await sync.cancelSync()

    expect(desktopClient.cancelSync).toHaveBeenCalledWith('123', 'nfe')
    expect(sync.isCancelling.value).toBe(true)
  })
})
