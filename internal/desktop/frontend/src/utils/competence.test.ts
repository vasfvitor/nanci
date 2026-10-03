import { describe, expect, it } from 'vitest'
import { competenceOf, isCompetence, shiftCompetence } from './competence'

describe('competence', () => {
  const now = new Date(2026, 8, 23)

  it('formats the local month of a date', () => {
    expect(competenceOf(now)).toBe('2026-09')
    expect(competenceOf(new Date(2026, 0, 31))).toBe('2026-01')
  })

  it('shifts across year boundaries', () => {
    expect(shiftCompetence('2026-01', -1, now)).toBe('2025-12')
    expect(shiftCompetence('2025-12', 1, now)).toBe('2026-01')
    expect(shiftCompetence('2026-06', 0, now)).toBe('2026-06')
  })

  it('starts an empty competence from the current month', () => {
    expect(shiftCompetence('', 1, now)).toBe('2026-10')
    expect(shiftCompetence(null, -1, now)).toBe('2026-08')
  })

  it('leaves an invalid competence unchanged', () => {
    expect(shiftCompetence('2026-13', 1, now)).toBe('2026-13')
    expect(shiftCompetence('abc', 1, now)).toBe('abc')
  })

  it('accepts only a YYYY-MM competence with a real month', () => {
    expect(isCompetence('2026-01')).toBe(true)
    expect(isCompetence('2026-12')).toBe(true)
    expect(isCompetence('2026-00')).toBe(false)
    expect(isCompetence('2026-13')).toBe(false)
    expect(isCompetence('2026-1')).toBe(false)
    expect(isCompetence('2026-0')).toBe(false)
    expect(isCompetence(' 2026-01')).toBe(false)
    expect(isCompetence('')).toBe(false)
    expect(isCompetence(null)).toBe(false)
    expect(isCompetence(undefined)).toBe(false)
  })
})
