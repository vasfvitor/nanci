import { describe, expect, it } from 'vitest'
import { UF_SIGLAS } from './uf'

describe('UF_SIGLAS', () => {
  it('lists the 27 federative units', () => {
    expect(UF_SIGLAS).toHaveLength(27)
  })

  it('has no repeated sigla', () => {
    expect(new Set(UF_SIGLAS).size).toBe(UF_SIGLAS.length)
  })

  it('is in alphabetical order', () => {
    expect([...UF_SIGLAS]).toEqual([...UF_SIGLAS].sort())
  })

  it('holds only two uppercase letters per sigla', () => {
    for (const uf of UF_SIGLAS) {
      expect(uf).toMatch(/^[A-Z]{2}$/)
    }
  })
})
