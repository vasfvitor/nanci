import { flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { shallowRef } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNSURecovery, type NSURecoverySource } from './useNSURecovery'
import { desktopClient } from '@/platform/wails/client'
import { useCompanySyncStore } from '@/stores/companySync'
import { useCTeDocumentsStore } from '@/stores/cteDocuments'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import { useWorkspaceStore } from '@/stores/workspace'
import { deferred } from '@/test/fixtures'
import type { NFeStatusResult, RecoverNSUResult } from '@/types/desktop'

type OkHandler = () => void

const notify = vi.fn()
const okHandlers: OkHandler[] = []
const dialog = vi.fn((_options: Record<string, unknown>) => ({
  onOk: (handler: OkHandler) => {
    okHandlers.push(handler)
  },
}))

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify, dialog }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listNFe: vi.fn(),
    statusNFe: vi.fn(),
    listNFePendingManifestacoes: vi.fn(),
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
    recoverNFeNSU: vi.fn(),
    recoverCTeNSU: vi.fn(),
  },
  wailsErrorCode: (error: { code?: string }) => error?.code ?? '',
  errorMessage: (error: { message?: string }) => error?.message ?? String(error),
}))

function status(overrides: Partial<NFeStatusResult> = {}): NFeStatusResult {
  return {
    NSUFaltantes: 6,
    RequestsLastHour: 4,
    RequestBudget: 20,
    NextAllowedAt: null,
    BlockedReason: '',
    ...overrides,
  } as NFeStatusResult
}

// recovery mounts useNSURecovery with the busy check the page composables
// pass: no company, or a sync of the company's source in flight.
function recovery(source: NSURecoverySource, blockedUntil: Date | null = null) {
  const syncStore = useCompanySyncStore()
  return useNSURecovery(source, {
    syncBlockedUntil: shallowRef(blockedUntil),
    syncBusy: (cnpj) => !cnpj || syncStore.isSyncing(cnpj, source),
  })
}

function result(overrides: Partial<RecoverNSUResult> = {}): RecoverNSUResult {
  return {
    Asked: 6,
    Recovered: 2,
    Empty: 4,
    Unreadable: 0,
    Expired: 0,
    Missing: 0,
    StopReason: '',
    NextAllowedAt: null,
    RequestsLastHour: 10,
    RequestBudget: 20,
    ...overrides,
  }
}

describe('useNSURecovery', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listNFe).mockResolvedValue([])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(status({ NSUFaltantes: 0 }))
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockResolvedValue([])
    vi.mocked(desktopClient.listCTe).mockResolvedValue([])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(status({ NSUFaltantes: 0 }) as never)
    useWorkspaceStore().cnpj = '123'
  })

  it('confirms what the run spends, recovers and refreshes the status', async () => {
    useNFeDocumentsStore().status = status()
    vi.mocked(desktopClient.recoverNFeNSU).mockResolvedValue(result())
    const nsu = recovery('nfe')
    expect(nsu.notice.value).toBe('6 NSUs faltando')
    expect(nsu.disabledReason.value).toBe('')

    nsu.confirmRecover()
    expect(dialog.mock.calls[0]?.[0]?.['message']).toBe(
      'Faltam 6 NSUs. A recuperação pede cada um à SEFAZ (consNSU) e usa até 6 das 20 consultas desta hora, ' +
        'deixando 2 para a sincronização. O que faltar continua na próxima vez.'
    )
    expect(desktopClient.recoverNFeNSU).not.toHaveBeenCalled()

    okHandlers[0]?.()
    await flushPromises()

    expect(desktopClient.recoverNFeNSU).toHaveBeenCalledWith('123')
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        message: '6 NSUs consultados, 2 com documento, 4 sem documento.',
      })
    )
    expect(desktopClient.statusNFe).toHaveBeenCalledWith('123')
  })

  it('keeps the recovery visible to a second instance while it is pending', async () => {
    useCTeDocumentsStore().status = status() as never
    const pending = deferred<RecoverNSUResult>()
    vi.mocked(desktopClient.recoverCTeNSU).mockReturnValue(pending.promise)

    const first = recovery('cte')
    const running = first.recover()

    const second = recovery('cte')
    expect(useCompanySyncStore().isSyncing('123', 'cte')).toBe(true)
    expect(useCompanySyncStore().canCancelSync('123', 'cte')).toBe(true)
    expect(second.disabledReason.value).toBe('Aguarde a operação em andamento terminar.')
    await expect(second.recover()).resolves.toBeNull()
    expect(desktopClient.recoverCTeNSU).toHaveBeenCalledTimes(1)

    pending.resolve(result({ Missing: 3, StopReason: 'limit' }))
    await running

    expect(useCompanySyncStore().isSyncing('123', 'cte')).toBe(false)
    expect(desktopClient.statusCTe).toHaveBeenCalledWith('123')
  })

  it('holds back while SEFAZ blocks consultas, but not for the wait after catching up', () => {
    const later = new Date(Date.now() + 30 * 60_000)
    const store = useNFeDocumentsStore()

    store.status = status({ NextAllowedAt: later, BlockedReason: 'caught_up' })
    expect(recovery('nfe', later).disabledReason.value).toBe('')

    store.status = status({ NextAllowedAt: later, BlockedReason: 'consumo_indevido' })
    expect(recovery('nfe', later).disabledReason.value).toBe(
      'Consultas bloqueadas no momento. Aguarde o horário indicado.'
    )

    store.status = status({ RequestsLastHour: 18 })
    const spent = recovery('nfe')
    expect(spent.disabledReason.value).toBe(
      'Sem consultas sobrando nesta hora; 2 ficam para a sincronização.'
    )
    spent.confirmRecover()
    expect(dialog).not.toHaveBeenCalled()
  })

  it('warns when SEFAZ blocked the CNPJ in the middle of the run', async () => {
    useNFeDocumentsStore().status = status()
    vi.mocked(desktopClient.recoverNFeNSU).mockResolvedValue(
      result({ Asked: 2, Recovered: 0, Empty: 1, Missing: 5, StopReason: 'consumo_indevido' })
    )

    await recovery('nfe').recover()

    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'warning',
        message:
          '2 NSUs consultados, 0 com documento, 1 sem documento. Ainda faltam 5 NSUs. ' +
          'A SEFAZ bloqueou o CNPJ por consumo indevido (656); tente depois do horário indicado.',
      })
    )
  })

  it('reports what a cancelled run did as a warning', async () => {
    useNFeDocumentsStore().status = status()
    vi.mocked(desktopClient.recoverNFeNSU).mockResolvedValue(
      result({ Asked: 2, Recovered: 1, Empty: 1, Missing: 4, StopReason: 'canceled' })
    )

    await recovery('nfe').recover()

    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'warning',
        message: expect.stringContaining('Recuperação cancelada'),
      })
    )
  })
})
