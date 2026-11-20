import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useImportActions, type ImportActionsOptions } from './useImportActions'
import ImportResultDialog from '@/components/ImportResultDialog.vue'
import type { ImportXMLResult } from '@/types/desktop'

const notify = vi.fn()
const dialog = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify, dialog }),
  useDialogPluginComponent: vi.fn(),
  copyToClipboard: vi.fn(),
}))

function importResult(rejected: number): ImportXMLResult {
  return { Files: [], Imported: 1, Updated: 0, Rejected: rejected }
}

function setup(source: 'nfe' | 'cte', overrides: Partial<ImportActionsOptions> = {}) {
  const refs = {
    cnpj: ref('123'),
    exporting: ref(false),
    isSyncing: ref(false),
    isResetting: ref(false),
  }
  const actions = useImportActions(source, {
    ...refs,
    importXML: async () => importResult(0),
    ...overrides,
  })
  return { refs, actions }
}

describe('useImportActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('reports an import without rejections and opens no dialog', async () => {
    const { actions } = setup('nfe')

    await actions.importXML()

    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: 'Importação: 1 importado, 0 atualizados, 0 rejeitados.',
    })
    expect(dialog).not.toHaveBeenCalled()
  })

  it.each(['nfe', 'cte'] as const)('lists the %s files when some were rejected', async (source) => {
    const result = importResult(1)
    const { actions } = setup(source, { importXML: async () => result })

    await actions.importXML()

    expect(notify).toHaveBeenCalledWith({
      type: 'warning',
      message: 'Importação: 1 importado, 0 atualizados, 1 rejeitado.',
    })
    expect(dialog).toHaveBeenCalledWith({
      component: ImportResultDialog,
      componentProps: { result, source },
    })
  })

  it('says nothing when the import did not run', async () => {
    const { actions } = setup('cte', { importXML: async () => null })

    await actions.importXML()

    expect(notify).not.toHaveBeenCalled()
  })

  it('reports a failed import', async () => {
    const { actions } = setup('cte', {
      importXML: async () => {
        throw new Error('boom')
      },
    })

    await actions.importXML()

    expect(notify).toHaveBeenCalledWith({ type: 'negative', message: 'Erro ao importar XML: boom' })
  })

  it('disables the import without a company and while the sync marker is held', () => {
    const { refs, actions } = setup('nfe')
    expect(actions.importDisabled.value).toBe(false)

    for (const marker of [refs.exporting, refs.isSyncing, refs.isResetting]) {
      marker.value = true
      expect(actions.importDisabled.value).toBe(true)
      marker.value = false
    }

    refs.cnpj.value = ''
    expect(actions.importDisabled.value).toBe(true)
  })
})
