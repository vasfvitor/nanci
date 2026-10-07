import { effectScope, nextTick, ref } from 'vue'
import { afterEach, beforeEach, expect, vi } from 'vitest'
import { type SefazBlockStatus, useSefazBlock } from './useSefazBlock'

// The tests run in America/Sao_Paulo (vitest.config.ts), so 13:00Z is 10:00.
const NOW = new Date('2026-10-07T13:00:00Z')

function blockStatus(fields: Partial<SefazBlockStatus> = {}): SefazBlockStatus {
  return {
    BlockedReason: 'consumo_indevido',
    RequestsLastHour: 0,
    RequestBudget: 20,
    NextAllowedAt: '2026-10-07T14:00:00Z',
    ...fields,
  }
}

function setup(initial: SefazBlockStatus | null) {
  const status = ref<SefazBlockStatus | null>(initial)
  const scope = effectScope()
  const block = scope.run(() => useSefazBlock(status))
  if (!block) throw new Error('useSefazBlock did not run')
  return { status, scope, ...block }
}

describe('useSefazBlock', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('is not blocked without a status, a wait time or with an invalid one', () => {
    for (const status of [null, blockStatus({ NextAllowedAt: undefined }), blockStatus({ NextAllowedAt: 'x' })]) {
      const { syncBlockedUntil, blockedText, scope } = setup(status)
      expect(syncBlockedUntil.value).toBeNull()
      expect(blockedText.value).toBe('')
      scope.stop()
    }
  })

  it('is not blocked when the wait time has passed', () => {
    const { syncBlockedUntil, blockedText } = setup(blockStatus({ NextAllowedAt: '2026-10-07T12:59:00Z' }))

    expect(syncBlockedUntil.value).toBeNull()
    expect(blockedText.value).toBe('')
  })

  it('explains a 656 block with the local time it ends', () => {
    const { syncBlockedUntil, blockedText } = setup(blockStatus())

    expect(syncBlockedUntil.value).toEqual(new Date('2026-10-07T14:00:00Z'))
    expect(blockedText.value).toBe('Consultas bloqueadas pela SEFAZ até 11:00 (cStat 656)')
  })

  it('shows the request budget of a rate block', () => {
    const { blockedText } = setup(
      blockStatus({ BlockedReason: 'rate_budget', RequestsLastHour: 20, NextAllowedAt: '2026-10-07T13:45:00Z' })
    )

    expect(blockedText.value).toBe('Limite de consultas por hora atingido (20/20); aguarde até 10:45')
  })

  it('clears itself once the block ends, without a status change', () => {
    const { syncBlockedUntil, blockedText } = setup(blockStatus())

    vi.advanceTimersByTime(60 * 60 * 1000)
    expect(syncBlockedUntil.value).not.toBeNull()

    vi.advanceTimersByTime(1000)
    expect(syncBlockedUntil.value).toBeNull()
    expect(blockedText.value).toBe('')
  })

  it('follows a new status and its new wait time', async () => {
    const { status, syncBlockedUntil, blockedText } = setup(blockStatus())

    status.value = blockStatus({ BlockedReason: 'caught_up', NextAllowedAt: '2026-10-07T15:00:00Z' })
    await nextTick()
    expect(blockedText.value).toBe('Sem novos documentos; próxima consulta a partir de 12:00')

    // The timer of the first wait time no longer clears the block.
    vi.advanceTimersByTime(60 * 60 * 1000 + 1000)
    expect(syncBlockedUntil.value).toEqual(new Date('2026-10-07T15:00:00Z'))

    vi.advanceTimersByTime(60 * 60 * 1000)
    expect(syncBlockedUntil.value).toBeNull()

    status.value = null
    await nextTick()
    expect(blockedText.value).toBe('')
  })

  it('stops its timer when the scope is disposed', () => {
    const { scope } = setup(blockStatus())
    expect(vi.getTimerCount()).toBe(1)

    scope.stop()
    expect(vi.getTimerCount()).toBe(0)
  })
})
