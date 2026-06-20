import { describe, expect, it } from 'vitest'
import {
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
})
