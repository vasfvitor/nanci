import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCTeDesacordoActions } from './useCTeDesacordoActions'
import CTeDesacordoDialog from '@/components/CTeDesacordoDialog.vue'
import { desktopClient, wailsErrorCode } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CTeDesacordoResult } from '@/types/desktop'
import { makeCTeRow } from '@/test/fixtures'

type OkHandler = (payload: unknown) => void

const notify = vi.fn()
const okHandlers: OkHandler[] = []
const dialog = vi.fn((_options: Record<string, unknown>) => ({
  onOk: (handler: OkHandler) => {
    okHandlers.push(handler)
  },
}))

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify, dialog }),
  useDialogPluginComponent: vi.fn(),
  copyToClipboard: vi.fn(),
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  wailsErrorCode: vi.fn(() => ''),
  desktopClient: {
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
    planCTeDesacordo: vi.fn(),
    registerCTeDesacordo: vi.fn(),
  },
}))

const row = makeCTeRow({ ChaveAcesso: 'a', TpAmb: '1', Modelo: '57', CompanyRole: 'tomador' })
const observacao = 'Carga entregue avariada no destino'

function result(status: CTeDesacordoResult['Status'], extra: Partial<CTeDesacordoResult> = {}) {
  return {
    ChaveAcesso: 'a',
    Status: status,
    CStat: '',
    XMotivo: '',
    Protocolo: '',
    RegisteredAt: null,
    ...extra,
  } as CTeDesacordoResult
}

async function openAndConfirm() {
  const actions = useCTeDesacordoActions()
  actions.openDesacordo(row)
  okHandlers[0]?.(observacao)
  await flushPromises()
  return actions
}

describe('useCTeDesacordoActions', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(wailsErrorCode).mockReturnValue('')
    vi.mocked(desktopClient.listCTe).mockResolvedValue([])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue({} as never)
    useWorkspaceStore().cnpj = '123'
  })

  it('opens the dialog with a plan loader for the row', async () => {
    useCTeDesacordoActions().openDesacordo(row)

    const options = dialog.mock.calls[0]?.[0] as {
      component: unknown
      componentProps: { row: unknown; load: (text: string) => Promise<unknown> }
    }
    expect(options.component).toBe(CTeDesacordoDialog)
    expect(options.componentProps.row).toBe(row)
    await options.componentProps.load(observacao)
    expect(desktopClient.planCTeDesacordo).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'a',
      Observacao: observacao,
    })
    expect(desktopClient.registerCTeDesacordo).not.toHaveBeenCalled()
  })

  it('sends after the dialog and tells the protocol', async () => {
    vi.mocked(desktopClient.registerCTeDesacordo).mockResolvedValue(
      result('registrada', { Protocolo: '135260000000123' })
    )
    await openAndConfirm()

    expect(desktopClient.registerCTeDesacordo).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'a',
      Observacao: observacao,
    })
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        message: 'Prestação em desacordo registrada. Protocolo 135260000000123.',
      })
    )
  })

  it('reports a rejection with the cStat', async () => {
    vi.mocked(desktopClient.registerCTeDesacordo).mockResolvedValue(
      result('rejeitada', { CStat: '787', XMotivo: 'Prazo excedido' })
    )
    await openAndConfirm()

    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'negative',
        message: 'Prestação em desacordo rejeitada pela SEFAZ: 787 - Prazo excedido',
      })
    )
  })

  it('warns when the password prompt was cancelled', async () => {
    vi.mocked(desktopClient.registerCTeDesacordo).mockRejectedValue(new Error('cancelada'))
    vi.mocked(wailsErrorCode).mockReturnValue('canceled')
    await openAndConfirm()

    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'warning',
        message: 'Envio da prestação em desacordo cancelado.',
      })
    )
  })
})
