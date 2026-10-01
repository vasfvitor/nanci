import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useDocumentListActions, type DocumentListActionsOptions } from './useDocumentListActions'

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

function setup(overrides: Partial<DocumentListActionsOptions> = {}) {
  const selected = ref<unknown[]>([])
  const scopeRows = ref([{ ChaveAcesso: 'a' }, { ChaveAcesso: 'b' }])
  const unviewedChaves = ref(['a', 'b'])
  const markViewed = vi.fn(async (chaves: string[]): Promise<number | null> => chaves.length)
  const exportList = vi.fn(async () => {})
  const actions = useDocumentListActions({
    source: 'nfse',
    selected,
    scopeRows,
    unviewedChaves,
    markViewed,
    exportFormats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
    exportList,
    ...overrides,
  })
  return { selected, scopeRows, unviewedChaves, markViewed, exportList, actions }
}

describe('useDocumentListActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    okHandlers.length = 0
  })

  it('marks the selection right away and reports the count', async () => {
    const { selected, markViewed, actions } = setup()
    selected.value = [{}]

    actions.confirmMarkViewed()
    await flushPromises()

    expect(dialog).not.toHaveBeenCalled()
    expect(markViewed).toHaveBeenCalledWith(['a', 'b'])
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '2 documentos marcados como vistos.' })
  })

  it('asks before marking the whole list, in the gender of the source', async () => {
    const { unviewedChaves, markViewed, actions } = setup({ source: 'cte' })
    unviewedChaves.value = ['a']

    actions.confirmMarkViewed()
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ message: 'Marcar como visto o CT-e novo da lista?' })
    )
    expect(markViewed).not.toHaveBeenCalled()

    okHandlers[0]?.(undefined)
    await flushPromises()
    expect(markViewed).toHaveBeenCalledWith(['a'])
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '1 documento marcado como visto.' })
  })

  it('does nothing without new documents', () => {
    const { unviewedChaves, markViewed, actions } = setup()
    unviewedChaves.value = []

    actions.confirmMarkViewed()

    expect(dialog).not.toHaveBeenCalled()
    expect(markViewed).not.toHaveBeenCalled()
  })

  it('asks in the plural for several new documents', () => {
    const { actions } = setup({ source: 'nfe' })

    actions.confirmMarkViewed()

    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ message: 'Marcar como vistas as 2 NF-e novas da lista?' })
    )
  })

  it('reports a failed marking under the source noun', async () => {
    const { selected, actions } = setup({
      markViewed: async () => {
        throw new Error('boom')
      },
    })
    selected.value = [{}]

    actions.confirmMarkViewed()
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({ type: 'negative', message: 'Erro ao marcar NFS-e como vistas: boom' })
  })

  it('opens the export dialog for the scope rows and exports the choice', async () => {
    const { exportList, actions } = setup({ source: 'nfe', showIncludeResumos: true })

    actions.openExportDialog()
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: {
          source: 'nfe',
          count: 2,
          scope: 'listed',
          formats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
          showIncludeResumos: true,
        },
      })
    )

    const choice = { format: 'zip', incremental: true, includeResumos: true }
    okHandlers[0]?.(choice)
    expect(exportList).toHaveBeenCalledWith(['a', 'b'], choice)
  })

  it('says the export is of the selection when there is one, and skips an empty scope', () => {
    const { selected, scopeRows, actions } = setup()
    selected.value = [{}]
    actions.openExportDialog()
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ componentProps: expect.objectContaining({ scope: 'selected' }) })
    )

    dialog.mockClear()
    scopeRows.value = []
    actions.openExportDialog()
    expect(dialog).not.toHaveBeenCalled()
  })
})
