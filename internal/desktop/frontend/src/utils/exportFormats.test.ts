import { describe, expect, it } from 'vitest'
import {
  CTE_EXPORT_FORMATS,
  exportedNoun,
  NFE_EXPORT_FORMATS,
  NFSE_EXPORT_FORMATS,
} from './exportFormats'

describe('exportFormats', () => {
  it('lists the formats of each source in the order the dialog shows them', () => {
    expect(NFSE_EXPORT_FORMATS.map((format) => format.value)).toEqual([
      'csv',
      'xlsx',
      'zip',
      'danfse-zip',
    ])
    expect(NFE_EXPORT_FORMATS.map((format) => format.value)).toEqual(['zip'])
    expect(CTE_EXPORT_FORMATS.map((format) => format.value)).toEqual(['zip'])
  })

  it('offers resumos only with the NF-e XMLs', () => {
    expect(NFSE_EXPORT_FORMATS.some((format) => format.resumos)).toBe(false)
    expect(CTE_EXPORT_FORMATS.some((format) => format.resumos)).toBe(false)
    expect(NFE_EXPORT_FORMATS).toEqual([{ label: 'XMLs (ZIP)', value: 'zip', resumos: true }])
  })

  it('names what each format writes', () => {
    expect(exportedNoun('danfse-zip')).toBe('DANFSe')
    expect(exportedNoun('danfe-zip')).toBe('DANFE')
    expect(exportedNoun('dacte-zip')).toBe('DACTE')
    expect(exportedNoun('zip')).toBe('XML')
    expect(exportedNoun('csv')).toBe('documento')
    expect(exportedNoun('xlsx')).toBe('documento')
    expect(exportedNoun('')).toBe('documento')
  })
})
