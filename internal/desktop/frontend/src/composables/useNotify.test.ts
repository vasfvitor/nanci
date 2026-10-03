import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { copyToClipboard } from 'quasar'
import { copyChave, useNotify } from './useNotify'
import { WailsClientError } from '@/platform/wails/client'

const notify = vi.hoisted(() => vi.fn())

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
  Notify: { create: notify },
  copyToClipboard: vi.fn(),
}))

describe('useNotify', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('prefixes error notifications with the action that failed', () => {
    useNotify().notifyError('Erro ao buscar', new Error('boom'))
    expect(notify).toHaveBeenCalledWith({ type: 'negative', message: 'Erro ao buscar: boom' })
  })

  it('warns about canceled, running and blocked syncs and reports other failures', () => {
    const { notifySyncError } = useNotify()

    notifySyncError('Erro na sincronização', new WailsClientError('senha não informada', 'canceled'))
    notifySyncError('Erro na sincronização', new WailsClientError('em andamento', 'sync_running'))
    notifySyncError('Erro na sincronização', { code: 'sefaz_blocked', message: 'bloqueado' })
    notifySyncError('Erro na sincronização', new Error('boom'))

    expect(notify.mock.calls.map(([options]) => options)).toEqual([
      { type: 'warning', message: 'Sincronização cancelada.' },
      { type: 'warning', message: 'Sincronização já em andamento para esta empresa.' },
      { type: 'warning', message: 'Consultas bloqueadas no momento. Aguarde o horário indicado.' },
      { type: 'negative', message: 'Erro na sincronização: boom' },
    ])
  })

  it('copies the chave as stored', async () => {
    vi.mocked(copyToClipboard).mockResolvedValue(undefined)
    await copyChave('35503082245852546000109000000000000126060000000011')
    expect(copyToClipboard).toHaveBeenCalledWith('35503082245852546000109000000000000126060000000011')
    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ type: 'positive' }))
  })

  it('ignores empty keys and reports clipboard failures', async () => {
    await copyChave('')
    expect(copyToClipboard).not.toHaveBeenCalled()

    vi.mocked(copyToClipboard).mockRejectedValue(new Error('negado'))
    await copyChave('35240912345678000199550010000123451123456789')
    await flushPromises()
    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao copiar chave: negado',
    })
  })

  it('notifies success, info and warning with extra options', () => {
    const { notifySuccess, notifyInfo, notifyWarning } = useNotify()

    notifySuccess('Feito.')
    notifyInfo('Nada mudou.', { timeout: 1000 })
    notifyWarning('Cuidado.', { caption: 'detalhe' })

    expect(notify.mock.calls.map(([options]) => options)).toEqual([
      { type: 'positive', message: 'Feito.' },
      { type: 'info', message: 'Nada mudou.', timeout: 1000 },
      { type: 'warning', message: 'Cuidado.', caption: 'detalhe' },
    ])
  })

  it('reports exports with one wording for every source', () => {
    const { notifyExported } = useNotify()

    notifyExported({ ExportedCount: 3, OutPath: 'C:/saida/notas.zip' })
    notifyExported({ ExportedCount: 1, OutPath: 'C:/saida/nota.zip' })
    notifyExported({ ExportedCount: 2, OutPath: 'C:/saida/danfse.zip' }, 'DANFSe')
    notifyExported({ ExportedCount: 0, OutPath: '' })
    notifyExported(null)

    expect(notify.mock.calls.map(([options]) => options)).toEqual([
      { type: 'positive', message: '3 XMLs exportados para C:/saida/notas.zip.' },
      { type: 'positive', message: '1 XML exportado para C:/saida/nota.zip.' },
      { type: 'positive', message: '2 DANFSes exportados para C:/saida/danfse.zip.' },
      { type: 'info', message: 'Nenhum documento para exportar.' },
    ])
  })
})
