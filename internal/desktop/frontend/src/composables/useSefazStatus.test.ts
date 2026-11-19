import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useSefazStatus } from './useSefazStatus'
import type { CompanySummary, SefazSourceStatus } from '@/types/desktop'

function status(overrides: Partial<SefazSourceStatus> = {}): SefazSourceStatus {
  return {
    CompanyName: 'Empresa da SEFAZ',
    TpAmb: '2',
    IdleDays: 0,
    NextAllowedAt: null,
    BlockedReason: '',
    ...overrides,
  } as SefazSourceStatus
}

describe('useSefazStatus', () => {
  it('names the company from the status, or from the list while it loads', () => {
    const current = ref<SefazSourceStatus | null>(null)
    const selected = ref({ Name: 'Empresa da lista' } as CompanySummary)
    const sefaz = useSefazStatus(current, selected)

    expect(sefaz.companyName.value).toBe('Empresa da lista')
    expect(sefaz.ambiente.value).toBeNull()

    current.value = status()
    expect(sefaz.companyName.value).toBe('Empresa da SEFAZ')
    expect(sefaz.ambiente.value?.label).toBe('Homologação')
  })

  it('warns after a long pause and says when the source is blocked', () => {
    const later = new Date(Date.now() + 30 * 60_000).toISOString()
    const current = ref<SefazSourceStatus | null>(
      status({ IdleDays: 50, NextAllowedAt: later, BlockedReason: 'consumo_indevido' })
    )
    const sefaz = useSefazStatus(current, ref(null))

    expect(sefaz.idleWarning.value).toBe('Sem consulta há 50 dias')
    expect(sefaz.syncBlockedUntil.value).toBeInstanceOf(Date)
    expect(sefaz.blockedText.value).not.toBe('')
  })
})
