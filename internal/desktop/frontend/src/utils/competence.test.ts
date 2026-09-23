import { describe, expect, it } from 'vitest'
import { competenceOf, shiftCompetence } from './competence'

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
})
