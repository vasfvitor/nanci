import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCTeDesacordo } from './useCTeDesacordo'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CTeDesacordoResult } from '@/types/desktop'
import { deferred } from '@/test/fixtures'

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: {
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
    planCTeDesacordo: vi.fn(),
    registerCTeDesacordo: vi.fn(),
  },
}))

const registered: CTeDesacordoResult = {
  ChaveAcesso: 'a',
  Status: 'registrada',
  CStat: '135',
  XMotivo: '',
  Protocolo: '135260000000123',
  RegisteredAt: null,
}

describe('useCTeDesacordo', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCTe).mockResolvedValue([])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue({} as never)
    useWorkspaceStore().cnpj = '123'
  })

  it('plans and sends for the workspace company, then refreshes the list', async () => {
    vi.mocked(desktopClient.registerCTeDesacordo).mockResolvedValue(registered)
    const desacordo = useCTeDesacordo()

    await desacordo.planDesacordo('a', 'Carga entregue avariada')
    await expect(desacordo.registerDesacordo('a', 'Carga entregue avariada')).resolves.toEqual(
      registered
    )

    const input = { CNPJ: '123', ChaveAcesso: 'a', Observacao: 'Carga entregue avariada' }
    expect(desktopClient.planCTeDesacordo).toHaveBeenCalledWith(input)
    expect(desktopClient.registerCTeDesacordo).toHaveBeenCalledWith(input)
    expect(desktopClient.listCTe).toHaveBeenCalled()
  })

  it('sends nothing without a company', async () => {
    useWorkspaceStore().cnpj = ''
    await expect(
      useCTeDesacordo().registerDesacordo('a', 'Carga entregue avariada')
    ).resolves.toBeNull()
    expect(desktopClient.registerCTeDesacordo).not.toHaveBeenCalled()
  })

  it('clears the in-flight marker when the send fails', async () => {
    vi.mocked(desktopClient.registerCTeDesacordo).mockRejectedValue(new Error('falhou'))
    const desacordo = useCTeDesacordo()
    await expect(desacordo.registerDesacordo('a', 'Carga entregue avariada')).rejects.toThrow(
      'falhou'
    )
    expect(desacordo.isSending('a')).toBe(false)
  })

  it('keeps a send visible to a second instance while it is pending', async () => {
    const pending = deferred<CTeDesacordoResult>()
    vi.mocked(desktopClient.registerCTeDesacordo).mockReturnValue(pending.promise)

    const first = useCTeDesacordo()
    const running = first.registerDesacordo('a', 'Carga entregue avariada')

    const second = useCTeDesacordo()
    expect(second.isSending('a')).toBe(true)
    await expect(second.registerDesacordo('a', 'Carga entregue avariada')).resolves.toBeNull()
    expect(desktopClient.registerCTeDesacordo).toHaveBeenCalledTimes(1)

    pending.resolve(registered)
    await running
    expect(second.isSending('a')).toBe(false)
  })
})
