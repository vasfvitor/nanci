import { describe, expect, it } from 'vitest'
import { cteLegend, nfeLegend, nfePendingLegend, nfseLegend, type LegendSection } from './stateLegends'
import {
  completenessFilterOptions,
  manifestacaoFilterOptions,
  nfeRoleFilterOptions,
  situacaoFilterOptions,
} from './nfeDisplay'
import { ctePapelFilterOptions, cteSituacaoFilterOptions } from './cteDisplay'

function section(sections: LegendSection[], title: string) {
  const found = sections.find((entry) => entry.title === title)
  if (!found) throw new Error(`section ${title} missing`)
  return found
}

function badges(sections: LegendSection[], title: string) {
  return section(sections, title).items.map((entry) => entry.badge)
}

// Every value a filter offers must be explained by the legend, in the same
// order and with the same label the table shows.
function filterLabels(options: { label: string; value: string }[]) {
  return options.filter((option) => option.value !== '').map((option) => option.label)
}

describe('stateLegends', () => {
  it('nfseLegend explains the new badge and every abbreviation', () => {
    const legend = nfseLegend()
    expect(badges(legend, 'Novo')).toEqual(['Novo'])
    expect(badges(legend, 'Direção')).toEqual(['P', 'T', 'I'])
    expect(badges(legend, 'Visibilidade')).toEqual(['PE', 'TE', 'IE', 'MR'])
    expect(badges(legend, 'Status')).toEqual(['N', 'C', 'S'])
    expect(section(legend, 'Novo').items[0]?.description).toContain('Marcar Vistos')
  })

  it('nfeLegend covers every NF-e filter value', () => {
    const legend = nfeLegend()
    expect(badges(legend, 'Situação')).toEqual(filterLabels(situacaoFilterOptions))
    expect(badges(legend, 'Completude')).toEqual(filterLabels(completenessFilterOptions))
    expect(badges(legend, 'Manifestação')).toEqual(filterLabels(manifestacaoFilterOptions))
    expect(badges(legend, 'Papel')).toEqual(filterLabels(nfeRoleFilterOptions))
    expect(badges(legend, 'Prazo')).toEqual(['Confirmada tacitamente'])
  })

  it('nfePendingLegend describes both pending groups with outlined chips', () => {
    const legend = nfePendingLegend()
    expect(legend.map((entry) => entry.title)).toEqual(['Sem ciência', 'Sem manifestação conclusiva'])
    for (const entry of legend) {
      expect(entry.items.every((item) => item.outline)).toBe(true)
    }
    expect(badges(legend, 'Sem manifestação conclusiva')).toContain('Confirmada tacitamente')
  })

  it('cteLegend covers every CT-e filter value and explains the export toggle', () => {
    const legend = cteLegend()
    expect(badges(legend, 'Papel')).toEqual(filterLabels(ctePapelFilterOptions))
    expect(badges(legend, 'Situação')).toEqual(filterLabels(cteSituacaoFilterOptions))
    expect(badges(legend, 'Documento')).toEqual(['CT-e', 'CT-e OS', 'GTV-e', 'CT-e Simplificado'])
    expect(section(legend, 'Somente novos').note).toContain('exportação')
  })

  it('never repeats a badge inside a section', () => {
    for (const legend of [nfseLegend(), nfeLegend(), nfePendingLegend(), cteLegend()]) {
      for (const entry of legend) {
        const seen = entry.items.map((item) => item.badge)
        expect(new Set(seen).size).toBe(seen.length)
      }
    }
  })
})
