import { describe, expect, it } from 'vitest'
import { mutedBannerClass, warningBannerClass } from './bannerClass'

describe('bannerClass', () => {
  it('colors a warning banner for each theme', () => {
    expect(warningBannerClass(false)).toBe('bg-orange-1 text-orange-10')
    expect(warningBannerClass(true)).toBe('bg-grey-9 text-orange-3')
  })

  it('colors a muted banner for each theme', () => {
    expect(mutedBannerClass(false)).toBe('bg-grey-2 text-grey-8')
    expect(mutedBannerClass(true)).toBe('bg-grey-9 text-grey-4')
  })
})
