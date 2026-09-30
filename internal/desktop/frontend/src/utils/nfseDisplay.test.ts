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
  describe('nfseStatus', () => {
    it('colors the status: normal positive, cancelada and substituida negative', () => {
      expect(nfseStatus.color('normal')).toBe('positive')
      expect(nfseStatus.color('cancelada')).toBe('negative')
      expect(nfseStatus.color('substituida')).toBe('negative')
    })
    it('colors an unknown status grey', () => {
      expect(nfseStatus.color('unknown')).toBe('grey')
      expect(nfseStatus.color('other')).toBe('grey')
    })
  })

  describe('nfseRole', () => {
    it('labels the known roles', () => {
      expect(nfseRole.label('prestada')).toBe('Prestada')
      expect(nfseRole.label('tomada')).toBe('Tomada')
      expect(nfseRole.label('intermediario')).toBe('Intermediário')
    })
    it('labels an unknown role as it is, and an empty one Desconhecido', () => {
      expect(nfseRole.label('foobar')).toBe('foobar')
      expect(nfseRole.label('')).toBe('Desconhecido')
    })
    it('colors the roles, grey by default', () => {
      expect(nfseRole.color('prestada')).toBe('primary')
      expect(nfseRole.color('tomada')).toBe('secondary')
      expect(nfseRole.color('intermediario')).toBe('accent')
      expect(nfseRole.color('none')).toBe('grey')
      expect(nfseRole.color('other')).toBe('grey')
    })
  })

  describe('nfseVisibility', () => {
    it('labels the known reasons', () => {
      expect(nfseVisibility.label('exact_prestador')).toBe('Prestador exato')
      expect(nfseVisibility.label('same_root_only')).toBe('Mesmo raiz apenas')
      expect(nfseVisibility.label('unknown')).toBe('Desconhecida')
    })
    it('labels an unknown reason as it is, and an empty one Desconhecido', () => {
      expect(nfseVisibility.label('other')).toBe('other')
      expect(nfseVisibility.label('')).toBe('Desconhecido')
    })
    it('colors the reasons, grey by default', () => {
      expect(nfseVisibility.color('exact_prestador')).toBe('positive')
      expect(nfseVisibility.color('exact_tomador')).toBe('positive')
      expect(nfseVisibility.color('same_root_only')).toBe('warning')
      expect(nfseVisibility.color('other')).toBe('grey')
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

    it('abbreviates the company without a fiscal role as SP', () => {
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
