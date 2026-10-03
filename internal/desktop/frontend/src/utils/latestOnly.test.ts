import { describe, expect, it } from 'vitest'
import { latestOnly } from './latestOnly'

describe('latestOnly', () => {
  it('keeps only the latest ticket current', () => {
    const gate = latestOnly()

    const first = gate.begin()
    expect(first()).toBe(true)

    const second = gate.begin()
    expect(first()).toBe(false)
    expect(second()).toBe(true)
  })

  it('keeps gates apart', () => {
    const a = latestOnly()
    const b = latestOnly()

    const ticket = a.begin()
    b.begin()

    expect(ticket()).toBe(true)
  })
})
