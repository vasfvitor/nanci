import { describe, expect, it } from 'vitest'
import { agree, DOCUMENT_SOURCES } from './documentSources'

describe('documentSources', () => {
  it('names each source with its article', () => {
    expect(DOCUMENT_SOURCES.nfse).toMatchObject({ noun: 'NFS-e', article: 'da' })
    expect(DOCUMENT_SOURCES.nfe).toMatchObject({ noun: 'NF-e', article: 'da' })
    expect(DOCUMENT_SOURCES.cte).toMatchObject({ noun: 'CT-e', article: 'do' })
  })

  it('agrees a word with the gender of the source and the count', () => {
    expect(agree('nfe', 'vist', 1)).toBe('vista')
    expect(agree('nfse', 'vist', 2)).toBe('vistas')
    expect(agree('cte', 'nov', 1)).toBe('novo')
    expect(agree('cte', 'nov', 3)).toBe('novos')
    expect(agree('cte', '', 2)).toBe('os')
  })
})
