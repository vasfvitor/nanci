import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeImportActions, type NFeImportActionsOptions } from './useNFeImportActions'
import ImportResultDialog from '@/components/ImportResultDialog.vue'
import NFeFetchDialog from '@/components/NFeFetchDialog.vue'
import type { FetchByChaveResult, ImportXMLResult, NFeStatusResult } from '@/types/desktop'

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

const chave = '35260912345678000195550010000000011000000010'

function importResult(rejected: number): ImportXMLResult {
  return { Files: [], Imported: 1, Updated: 0, Rejected: rejected }
}

function fetchResult(overrides: Partial<FetchByChaveResult>): FetchByChaveResult {
  return {
    ChaveAcesso: chave,
    Outcome: 'completa',
    Message: '',
    RequestsLastHour: 5,
    RequestBudget: 20,
    ...overrides,
  }
}

function setup(overrides: Partial<NFeImportActionsOptions> = {}) {
  const importXML = vi.fn(async (): Promise<ImportXMLResult | null> => importResult(0))
  const fetchByChave = vi.fn(async (_chave: string): Promise<FetchByChaveResult | null> =>
    fetchResult({})
  )
  const actions = useNFeImportActions({
    cnpj: ref('123'),
    status: ref({ RequestsLastHour: 4, RequestBudget: 20 } as NFeStatusResult),
    importXML,
    fetchByChave,
    ...overrides,
  })
  return { importXML, fetchByChave, actions }
}

describe('useNFeImportActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    okHandlers.length = 0
  })

  it('reports an import without rejections and opens no dialog', async () => {
    const { actions } = setup()

    await actions.importXML()

    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: 'Importação: 1 importado, 0 atualizados, 0 rejeitados.',
    })
    expect(dialog).not.toHaveBeenCalled()
  })

  it('lists the files when some were rejected', async () => {
    const result = importResult(1)
    const { actions } = setup({ importXML: async () => result })

    await actions.importXML()

    expect(notify).toHaveBeenCalledWith({
      type: 'warning',
      message: 'Importação: 1 importado, 0 atualizados, 1 rejeitado.',
    })
    expect(dialog).toHaveBeenCalledWith({
      component: ImportResultDialog,
      componentProps: { result, source: 'nfe' },
    })
  })

  it('says nothing when the import did not run', async () => {
    const { actions } = setup({ importXML: async () => null })

    await actions.importXML()

    expect(notify).not.toHaveBeenCalled()
  })

  it('reports a failed import', async () => {
    const { actions } = setup({
      importXML: async () => {
        throw new Error('boom')
      },
    })

    await actions.importXML()

    expect(notify).toHaveBeenCalledWith({ type: 'negative', message: 'Erro ao importar XML: boom' })
  })

  it('opens no fetch dialog without a company', () => {
    const { actions } = setup({ cnpj: ref('') })

    actions.openFetchDialog()

    expect(dialog).not.toHaveBeenCalled()
  })

  it('opens the fetch dialog with the hourly budget and fetches the chave typed', async () => {
    const { fetchByChave, actions } = setup()

    actions.openFetchDialog()
    expect(dialog).toHaveBeenCalledWith({
      component: NFeFetchDialog,
      componentProps: { requestsLastHour: 4, requestBudget: 20 },
    })
    expect(fetchByChave).not.toHaveBeenCalled()

    okHandlers[0]?.(chave)
    await flushPromises()

    expect(fetchByChave).toHaveBeenCalledWith(chave)
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: 'NF-e completa salva.' })
  })

  it('opens the fetch dialog with an empty budget before the status loads', () => {
    const { actions } = setup({ status: ref(null) })

    actions.openFetchDialog()

    expect(dialog).toHaveBeenCalledWith({
      component: NFeFetchDialog,
      componentProps: { requestsLastHour: 0, requestBudget: 0 },
    })
  })

  it.each([
    [
      'resumo',
      'a SEFAZ entregou só o resumo',
      { type: 'info', message: 'A SEFAZ entregou só o resumo.' },
    ],
    ['resumo', '', { type: 'info', message: 'A SEFAZ entregou só o resumo da NF-e.' }],
    ['not_found', 'NF-e não localizada', { type: 'warning', message: 'NF-e não localizada.' }],
    ['not_found', '', { type: 'warning', message: 'A SEFAZ não encontrou a NF-e.' }],
  ] as const)('reports a %s fetch with the message %j', async (outcome, message, notice) => {
    const { actions } = setup({
      fetchByChave: async () => fetchResult({ Outcome: outcome, Message: message }),
    })

    actions.openFetchDialog()
    okHandlers[0]?.(chave)
    await flushPromises()

    expect(notify).toHaveBeenCalledWith(notice)
  })

  it('reports a failed fetch as a sync error', async () => {
    const { actions } = setup({
      fetchByChave: async () => {
        throw new Error('chave de acesso inválida')
      },
    })

    actions.openFetchDialog()
    okHandlers[0]?.(chave)
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao buscar a NF-e: chave de acesso inválida',
    })
  })
})
