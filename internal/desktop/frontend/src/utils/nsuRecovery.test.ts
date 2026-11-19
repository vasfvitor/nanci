import { describe, expect, it } from 'vitest'
import { missingNSUsText, recoveryAllowance, recoverySummary } from './nsuRecovery'
import type { RecoverNSUResult } from '@/types/desktop'

function result(overrides: Partial<RecoverNSUResult> = {}): RecoverNSUResult {
  return {
    Asked: 0,
    Recovered: 0,
    Empty: 0,
    Unreadable: 0,
    Expired: 0,
    Missing: 0,
    StopReason: '',
    RequestsLastHour: 0,
    RequestBudget: 20,
    ...overrides,
  }
}

describe('nsuRecovery', () => {
  it('asks for the missing NSUs up to the hour left after the reserve', () => {
    expect(recoveryAllowance(null)).toBe(0)
    expect(recoveryAllowance({ NSUFaltantes: 3, RequestsLastHour: 0, RequestBudget: 20 })).toBe(3)
    expect(recoveryAllowance({ NSUFaltantes: 50, RequestsLastHour: 5, RequestBudget: 20 })).toBe(13)
    expect(recoveryAllowance({ NSUFaltantes: 50, RequestsLastHour: 19, RequestBudget: 20 })).toBe(0)
  })

  it('names one NSU and many', () => {
    expect(missingNSUsText(1)).toBe('1 NSU faltando')
    expect(missingNSUsText(12)).toBe('12 NSUs faltando')
  })

  it('says a cancelled run keeps what it did', () => {
    expect(
      recoverySummary(
        result({ Asked: 2, Recovered: 1, Empty: 1, Missing: 3, StopReason: 'canceled' })
      )
    ).toBe(
      '2 NSUs consultados, 1 com documento, 1 sem documento. Ainda faltam 3 NSUs. ' +
        'Recuperação cancelada; o que falta continua na próxima vez.'
    )
  })

  it('sums up a run', () => {
    expect(recoverySummary(result())).toBe('Nenhum NSU faltando.')
    expect(recoverySummary(result({ Asked: 0, Expired: 4 }))).toBe(
      '0 NSUs consultados, 0 com documento, 0 sem documento, 4 fora da janela de 90 dias.'
    )
    expect(
      recoverySummary(
        result({
          Asked: 3,
          Recovered: 1,
          Empty: 1,
          Unreadable: 1,
          Missing: 1,
          StopReason: 'rate_budget',
        })
      )
    ).toBe(
      '3 NSUs consultados, 1 com documento, 1 sem documento, 1 ilegíveis (veja o log). ' +
        'Ainda falta 1 NSU. As consultas desta hora acabaram.'
    )
  })
})
