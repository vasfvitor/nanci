import { describe, expect, it } from 'vitest'
import type { DisplayTable } from './sefazDisplay'
import {
  nfseAmbiente,
  nfseEvent,
  nfseRole,
  nfseStateBadges,
  nfseStatus,
  nfseStatusLine,
  nfseVisibility,
  statusColor,
  roleLabel,
  roleColor,
  visibilityLabel,
  visibilityColor,
  getRoleAbbreviation,
  getVisibilityAbbreviation,
  getStatusAbbreviation,
  getStatusLabel,
} from './nfseDisplay'

// expectShortUniqueAbbrs checks that every value of table has an
// abbreviation of one or two characters, unique within the table.
function expectShortUniqueAbbrs(table: DisplayTable) {
  const abbrs = table.values().map((value) => table.abbr(value))
  for (const abbr of abbrs) {
    expect(abbr).toMatch(/^.{1,2}$/u)
  }
  expect(new Set(abbrs).size).toBe(abbrs.length)
}

describe('nfseDisplay utility functions', () => {
  describe('statusColor', () => {
    it('returns positive for normal status', () => {
      expect(statusColor('normal')).toBe('positive')
    })
    it('returns negative for cancelada or substituida', () => {
      expect(statusColor('cancelada')).toBe('negative')
      expect(statusColor('substituida')).toBe('negative')
    })
    it('returns grey for unknown status', () => {
      expect(statusColor('unknown')).toBe('grey')
    })
  })

  describe('roleLabel', () => {
    it('returns translated label for known roles', () => {
      expect(roleLabel('prestada')).toBe('Prestada')
      expect(roleLabel('tomada')).toBe('Tomada')
      expect(roleLabel('intermediario')).toBe('Intermediário')
    })
    it('returns the input role or Desconhecido if unknown', () => {
      expect(roleLabel('foobar')).toBe('foobar')
      expect(roleLabel('')).toBe('Desconhecido')
    })
  })

  describe('roleColor', () => {
    it('returns correct color for roles', () => {
      expect(roleColor('prestada')).toBe('primary')
      expect(roleColor('tomada')).toBe('secondary')
      expect(roleColor('intermediario')).toBe('accent')
      expect(roleColor('none')).toBe('grey')
    })
    it('defaults to grey', () => {
      expect(roleColor('other')).toBe('grey')
    })
  })

  describe('visibilityLabel', () => {
    it('returns translated label for known reasons', () => {
      expect(visibilityLabel('exact_prestador')).toBe('Prestador exato')
      expect(visibilityLabel('same_root_only')).toBe('Mesmo raiz apenas')
    })
    it('returns original or Desconhecida if unknown', () => {
      expect(visibilityLabel('other')).toBe('other')
      expect(visibilityLabel('')).toBe('Desconhecida')
    })
  })

  describe('visibilityColor', () => {
    it('returns correct color for known reasons', () => {
      expect(visibilityColor('exact_prestador')).toBe('positive')
      expect(visibilityColor('same_root_only')).toBe('warning')
    })
    it('defaults to grey', () => {
      expect(visibilityColor('other')).toBe('grey')
    })
  })

  describe('getRoleAbbreviation', () => {
    it('returns correct abbreviation', () => {
      expect(getRoleAbbreviation('prestada')).toBe('P')
      expect(getRoleAbbreviation('tomada')).toBe('T')
      expect(getRoleAbbreviation('intermediario')).toBe('I')
      expect(getRoleAbbreviation('other')).toBe('-')
      expect(getRoleAbbreviation()).toBe('-')
    })
  })

  describe('getVisibilityAbbreviation', () => {
    it('returns correct abbreviation', () => {
      expect(getVisibilityAbbreviation('exact_prestador')).toBe('PE')
      expect(getVisibilityAbbreviation('exact_tomador')).toBe('TE')
      expect(getVisibilityAbbreviation('exact_intermediario')).toBe('IE')
      expect(getVisibilityAbbreviation('same_root_only')).toBe('MR')
      expect(getVisibilityAbbreviation('other')).toBe('?')
      expect(getVisibilityAbbreviation()).toBe('?')
    })
  })

  describe('getStatusAbbreviation', () => {
    it('returns correct abbreviation', () => {
      expect(getStatusAbbreviation('normal')).toBe('N')
      expect(getStatusAbbreviation('cancelada')).toBe('C')
      expect(getStatusAbbreviation('substituida')).toBe('S')
      expect(getStatusAbbreviation('other')).toBe('?')
      expect(getStatusAbbreviation()).toBe('?')
    })
  })

  describe('getStatusLabel', () => {
    it('returns correct label', () => {
      expect(getStatusLabel('normal')).toBe('Normal')
      expect(getStatusLabel('cancelada')).toBe('Cancelada')
      expect(getStatusLabel('substituida')).toBe('Substituída')
      expect(getStatusLabel('other')).toBe('other')
      expect(getStatusLabel()).toBe('Desconhecido')
    })
  })

  describe('display tables', () => {
    it('abbreviate every value with one or two unique characters', () => {
      for (const table of [nfseStatus, nfseVisibility, nfseRole]) {
        expectShortUniqueAbbrs(table)
      }
    })

    it('keep the abbreviations the NFS-e page already shows', () => {
      expect(nfseStatus.values().map(nfseStatus.abbr)).toEqual(['N', 'C', 'S'])
      expect(nfseVisibility.values().map(nfseVisibility.abbr)).toEqual(['PE', 'TE', 'IE', 'MR', '?'])
      expect(nfseRole.values().map(nfseRole.abbr)).toEqual(['P', 'T', 'I', 'SP'])
    })

    it('keeps the old role abbreviation fallback for none', () => {
      expect(getRoleAbbreviation('none')).toBe('-')
      expect(nfseRole.abbr('none')).toBe('SP')
    })

    it('colors the event types the ADN sends', () => {
      expect(nfseEvent.label('cancelamento')).toBe('Cancelamento')
      expect(nfseEvent.color('cancelamento')).toBe('negative')
      expect(nfseEvent.label('substituicao')).toBe('Substituição')
      expect(nfseEvent.color('substituicao')).toBe('warning')
      expect(nfseEvent.color('unknown')).toBe('grey')
    })
  })

  describe('nfseStateBadges', () => {
    it('lists status, visibilidade and papel in order', () => {
      const badges = nfseStateBadges({
        Status: 'cancelada',
        VisibilityReason: 'same_root_only',
        CompanyRole: 'tomada',
      })
      expect(badges.map((badge) => badge.kind)).toEqual(['Status', 'Visibilidade', 'Papel'])
      expect(badges.map((badge) => badge.abbr)).toEqual(['C', 'MR', 'T'])
      expect(badges[0]).toEqual({
        key: 'Status:cancelada',
        abbr: 'C',
        label: 'Cancelada',
        color: 'negative',
        kind: 'Status',
      })
      expect(badges.some((badge) => badge.secondary)).toBe(false)
    })
  })

  describe('nfseAmbiente', () => {
    it('names and colors the ADN environment like the SEFAZ tpAmb', () => {
      expect(nfseAmbiente('producao')).toEqual({ label: 'Produção', color: 'negative' })
      expect(nfseAmbiente('producao_restrita')).toEqual({
        label: 'Produção restrita',
        color: 'warning',
      })
      expect(nfseAmbiente('')).toEqual({ label: 'Ambiente desconhecido', color: 'grey' })
    })
  })

  describe('nfseStatusLine', () => {
    it('sums up the last sync and NSU', () => {
      const line = nfseStatusLine({ LastSyncAt: '2026-01-02T03:04:05Z', LastFoundNSU: 42 })
      expect(line).toMatch(/^Última sincronização: .+ · NSU 42$/u)
      expect(line).not.toContain('nunca')
    })

    it('says when the company never synced', () => {
      expect(nfseStatusLine({ LastFoundNSU: null })).toBe('Última sincronização: nunca · NSU —')
    })
  })
})
