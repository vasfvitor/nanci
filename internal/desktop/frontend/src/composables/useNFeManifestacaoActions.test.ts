import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeManifestacaoActions } from './useNFeManifestacaoActions'
import NFeCienciaConfirmDialog from '@/components/NFeCienciaConfirmDialog.vue'
import NFeEventResultsDialog from '@/components/NFeEventResultsDialog.vue'
import NFeManifestacaoDialog from '@/components/NFeManifestacaoDialog.vue'
import { desktopClient, wailsErrorCode } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import type { NFeCienciaPlan, NFeEventBatchResult, NFeEventResult, NFeRow } from '@/types/desktop'
import { makeNFeRow } from '@/test/fixtures'

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
    listNFe: vi.fn(),
    statusNFe: vi.fn(),
    listNFePendingManifestacoes: vi.fn(),
    planNFeCiencia: vi.fn(),
    registerNFeCiencia: vi.fn(),
    registerNFeManifestacao: vi.fn(),
  },
}))

function nfeRow(chave: string, overrides: Partial<NFeRow> = {}): NFeRow {
  return makeNFeRow({ ChaveAcesso: chave, CompanyRole: 'destinatario', ...overrides })
}

function outcome(status: NFeEventResult['Status'], extra: Partial<NFeEventResult> = {}) {
  return {
    ChaveAcesso: 'a',
    TpEvento: '210210',
    Status: status,
    CStat: '',
    XMotivo: '',
    Protocolo: '',
    ...extra,
  } as NFeEventResult
}

function setup() {
  const syncNFe = vi.fn(async () => {})
  const actions = useNFeManifestacaoActions({
    cnpj: ref('123'),
    companyName: ref('Empresa Um'),
    tpAmb: ref('2'),
    syncNFe,
  })
  return { syncNFe, actions }
}

describe('useNFeManifestacaoActions', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(wailsErrorCode).mockReturnValue('')
    vi.mocked(desktopClient.listNFe).mockResolvedValue([])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue({} as never)
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockResolvedValue([])
    useWorkspaceStore().cnpj = '123'
  })

  it('plans no ciência for a selection without eligible notes', () => {
    useNFeDocumentsStore().selected = [nfeRow('b', { CienciaBlockReason: 'emitida' })]
    const { actions } = setup()

    actions.registerSelectedCiencia()

    expect(desktopClient.planNFeCiencia).not.toHaveBeenCalled()
  })

  it('plans the whole selection and confirms the eligible notes before sending', async () => {
    const blocked = nfeRow('b', { CienciaBlockReason: 'emitida' })
    useNFeDocumentsStore().selected = [nfeRow('a'), blocked]
    const plan: NFeCienciaPlan = {
      Eligible: [nfeRow('a')],
      Skipped: [{ ChaveAcesso: 'b', Reason: 'emitida' }],
    }
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue(plan)
    vi.mocked(desktopClient.registerNFeCiencia).mockResolvedValue({
      Results: [outcome('registrada')],
      Skipped: [],
      Interrupted: '',
    })
    const { syncNFe, actions } = setup()

    actions.registerSelectedCiencia()
    await flushPromises()

    expect(desktopClient.planNFeCiencia).toHaveBeenCalledWith('123', ['a', 'b'])
    expect(dialog).toHaveBeenCalledWith({
      component: NFeCienciaConfirmDialog,
      componentProps: { companyName: 'Empresa Um', cnpj: '123', tpAmb: '2', plan },
    })
    expect(desktopClient.registerNFeCiencia).not.toHaveBeenCalled()

    okHandlers[0]?.(['a'])
    await flushPromises()

    expect(desktopClient.registerNFeCiencia).toHaveBeenCalledWith('123', ['a'])
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        message: 'Ciência: 1 registradas, 0 já registradas, 0 rejeitadas, 0 não enviadas.',
        caption: 'O XML completo chega na próxima sincronização.',
      })
    )
    expect(dialog).toHaveBeenCalledTimes(1)

    const notice = notify.mock.calls[0]?.[0] as { actions: { handler: () => void }[] }
    notice.actions[0]?.handler()
    expect(syncNFe).toHaveBeenCalledTimes(1)
  })

  it('warns and opens no dialog when the plan has no eligible note', async () => {
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue({
      Eligible: [],
      Skipped: [{ ChaveAcesso: 'a', Reason: 'ciência já registrada' }],
    })
    const { actions } = setup()

    await actions.startCiencia(['a'])

    expect(dialog).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ type: 'warning' }))
  })

  it('reports a failed plan', async () => {
    vi.mocked(desktopClient.planNFeCiencia).mockRejectedValue(new Error('boom'))
    const { actions } = setup()

    await actions.startCiencia(['a'])

    expect(dialog).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao preparar a ciência: boom',
    })
  })

  it('shows the per-note results when a ciência has problems', async () => {
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue({
      Eligible: [nfeRow('a')],
      Skipped: [],
    })
    const result: NFeEventBatchResult = {
      Results: [outcome('rejeitada', { CStat: '573', XMotivo: 'Duplicidade' })],
      Skipped: [],
      Interrupted: '',
    }
    vi.mocked(desktopClient.registerNFeCiencia).mockResolvedValue(result)
    const { actions } = setup()

    await actions.startCiencia(['a'])
    okHandlers[0]?.(['a'])
    await flushPromises()

    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ type: 'warning' }))
    expect(dialog).toHaveBeenLastCalledWith({
      component: NFeEventResultsDialog,
      componentProps: { result },
    })
  })

  it('says when the ciência was canceled', async () => {
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue({
      Eligible: [nfeRow('a')],
      Skipped: [],
    })
    vi.mocked(desktopClient.registerNFeCiencia).mockRejectedValue(new Error('canceled'))
    vi.mocked(wailsErrorCode).mockReturnValue('canceled')
    const { actions } = setup()

    await actions.startCiencia(['a'])
    okHandlers[0]?.(['a'])
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({ type: 'warning', message: 'Envio de ciência cancelado.' })
  })

  it.each([
    {
      status: 'registrada' as const,
      notice: { type: 'positive', message: 'Manifestação registrada. Protocolo 135.' },
    },
    {
      status: 'ja_registrada' as const,
      notice: { type: 'info', message: 'A manifestação já estava registrada na SEFAZ.' },
    },
    {
      status: 'rejeitada' as const,
      notice: {
        type: 'negative',
        message: 'Manifestação rejeitada pela SEFAZ: 573 - Duplicidade',
      },
    },
    {
      status: 'nao_enviada' as const,
      notice: { type: 'warning', message: 'Manifestação não enviada. 573 - Duplicidade' },
    },
  ])('sends the manifestação the dialog returns and reports $status', async (c) => {
    const note = nfeRow('a')
    const extra =
      c.status === 'registrada' ? { Protocolo: '135' } : { CStat: '573', XMotivo: 'Duplicidade' }
    vi.mocked(desktopClient.registerNFeManifestacao).mockResolvedValue(outcome(c.status, extra))
    const { actions } = setup()

    actions.openManifestacao(note)
    expect(dialog).toHaveBeenCalledWith({
      component: NFeManifestacaoDialog,
      componentProps: { note, tpAmb: '2' },
    })

    okHandlers[0]?.({ tipo: '210240', justificativa: 'mercadoria não entregue' })
    await flushPromises()

    expect(desktopClient.registerNFeManifestacao).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'a',
      Tipo: '210240',
      Justificativa: 'mercadoria não entregue',
    })
    expect(notify).toHaveBeenCalledWith(c.notice)
  })

  it('warns when the note already has an event being sent', async () => {
    useNFeDocumentsStore().manifestacaoInFlight.add('a')
    const { actions } = setup()

    actions.openManifestacao(nfeRow('a'))
    okHandlers[0]?.({ tipo: '210200', justificativa: '' })
    await flushPromises()

    expect(desktopClient.registerNFeManifestacao).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith({
      type: 'warning',
      message: 'Esta nota já tem um envio em andamento.',
    })
  })
})
