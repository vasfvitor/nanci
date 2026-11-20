import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCTeLoaders } from './useCTeLoaders'
import { desktopClient } from '@/platform/wails/client'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CTeRow, CTeStatusResult } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
  },
}))

const row = { ChaveAcesso: 'a' } as CTeRow
const statusResult = { CNPJ: '123' } as CTeStatusResult

describe('useCTeLoaders', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCTe).mockResolvedValue([row])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(statusResult)
    useWorkspaceStore().cnpj = '123'
  })

  it('records a focused search under the workspace key, so a remount keeps its rows', async () => {
    useWorkspaceStore().competence = '2026-05'
    const store = useCTeDocumentsStore()
    store.focusChave('CTE-KEY')

    await useCTeLoaders().search()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ Competence: '', ChavesAcesso: ['CTE-KEY'] })
    )
    expect(store.rowsFor).toEqual({ cnpj: '123', competence: '2026-05' })
  })

  it('refreshes the CT-e list and status of the selected company', async () => {
    const store = useCTeDocumentsStore()

    await useCTeLoaders().refresh('123')

    expect(desktopClient.listCTe).toHaveBeenCalledWith(store.listInput)
    expect(store.rows).toEqual([row])
    expect(store.status).toEqual(statusResult)
    expect(store.loading).toBe(false)
  })

  it('does not send an NF-e key that is not 44 characters', async () => {
    const store = useCTeDocumentsStore()
    store.filter.NFeChave = '3526'

    await expect(useCTeLoaders().search()).resolves.toEqual([])
    expect(desktopClient.listCTe).not.toHaveBeenCalled()

    store.filter.NFeChave = '3'.repeat(44)
    await useCTeLoaders().search()
    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ NFeChave: '3'.repeat(44) })
    )
  })

  it('lists only the unviewed CT-e when "Somente não vistos" is on', async () => {
    useCTeDocumentsStore().filter.OnlyUnread = true

    await useCTeLoaders().search()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ OnlyUnread: true })
    )
  })
})
