import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { readStorage, writeStorage } from './storage'

describe('storage', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('reads, writes and removes a key', () => {
    expect(readStorage('nanci:test')).toBeNull()

    writeStorage('nanci:test', 'x')
    expect(readStorage('nanci:test')).toBe('x')
    expect(localStorage.getItem('nanci:test')).toBe('x')

    writeStorage('nanci:test', null)
    expect(readStorage('nanci:test')).toBeNull()
  })

  it('returns null and swallows errors when storage throws', () => {
    localStorage.setItem('nanci:test', 'x')
    vi.spyOn(globalThis, 'localStorage', 'get').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })

    expect(() => localStorage).toThrow('denied')
    expect(readStorage('nanci:test')).toBeNull()
    expect(() => writeStorage('nanci:test', 'x')).not.toThrow()
    expect(() => writeStorage('nanci:test', null)).not.toThrow()
  })
})
