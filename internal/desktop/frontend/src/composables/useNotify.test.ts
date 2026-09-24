import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { copyToClipboard } from 'quasar'
import { useNotify } from './useNotify'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
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

    notifySyncError('Erro na sincronização', new Error('ERR_CANCELED: senha não informada'))
    notifySyncError('Erro na sincronização', new Error('ERR_SYNC_RUNNING: em andamento'))
    notifySyncError('Erro na sincronização', new Error('ERR_SEFAZ_BLOCKED: bloqueado'))
    notifySyncError('Erro na sincronização', new Error('boom'))

    expect(notify.mock.calls.map(([options]) => options)).toEqual([
      { type: 'warning', message: 'Sincronização cancelada.' },
      { type: 'warning', message: 'Sincronização já em andamento para esta empresa.' },
      { type: 'warning', message: 'Consultas bloqueadas no momento. Aguarde o horário indicado.' },
      { type: 'negative', message: 'Erro na sincronização: boom' },
    ])
  })

  it('copies access keys without the NFS prefix', async () => {
    vi.mocked(copyToClipboard).mockResolvedValue(undefined)
    await useNotify().copyChave('NFS123')
    expect(copyToClipboard).toHaveBeenCalledWith('123')
    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ type: 'positive' }))
  })

  it('ignores empty keys and reports clipboard failures', async () => {
    const { copyChave } = useNotify()
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
})
