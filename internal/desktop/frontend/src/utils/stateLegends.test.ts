import { describe, expect, it } from 'vitest'
import {
  CTE_LEGEND,
  NFE_LEGEND,
  NFE_PENDING_LEGEND,
  NFSE_LEGEND,
  type LegendSection,
} from './stateLegends'
import {
  completenessFilterOptions,
  manifestacaoFilterOptions,
  nfeRoleFilterOptions,
  nfeStateBadges,
  situacaoFilterOptions,
} from './nfeDisplay'
import { ctePapelFilterOptions, cteSituacaoFilterOptions, cteStateBadges } from './cteDisplay'
import { nfseStateBadges } from './nfseDisplay'

function section(sections: LegendSection[], title: string) {
  const found = sections.find((entry) => entry.title === title)
  if (!found) throw new Error(`section ${title} missing`)
  return found
}

function badges(sections: LegendSection[], title: string) {
  return section(sections, title).items.map((entry) => entry.badge)
}

function names(sections: LegendSection[], title: string) {
  return section(sections, title).items.map((entry) => entry.name)
}

// Every value a filter offers must be explained by the legend, in the same
// order and with the same label the table shows.
function filterLabels(options: { label: string; value: string }[]) {
  return options.filter((option) => option.value !== '').map((option) => option.label)
}

describe('stateLegends', () => {
  it('NFSE_LEGEND explains the new badge and every abbreviation', () => {
    const legend = NFSE_LEGEND
    expect(badges(legend, 'Novo')).toEqual(['Novo'])
    expect(badges(legend, 'Papel')).toEqual(['P', 'T', 'I', 'SP'])
    expect(badges(legend, 'Visibilidade')).toEqual(['PE', 'TE', 'IE', 'MR'])
    expect(badges(legend, 'Status')).toEqual(['N', 'C', 'S'])
    expect(section(legend, 'Status').items[1]).toMatchObject({
      badge: 'C',
      name: 'Cancelada',
      color: 'negative',
    })
  })

  it('NFSE_LEGEND lists the groups in the order of the badges', () => {
    const titles = NFSE_LEGEND.map((entry) => entry.title)
    const kinds = nfseStateBadges({ Status: '', VisibilityReason: '', CompanyRole: '' }).map(
      (badge) => badge.kind
    )
    expect(titles).toEqual(['Novo', ...kinds])
  })

  it('every legend explains the new badge the same way', () => {
    const novo = section(NFSE_LEGEND, 'Novo')
    expect(novo.items[0]?.description).toContain('"Marcar vistos"')
    expect(novo.items[0]?.description).toContain('"Somente não vistos"')
    for (const legend of [NFSE_LEGEND, NFE_LEGEND, CTE_LEGEND]) {
      expect(section(legend, 'Novo')).toEqual(novo)
    }
  })

  it('NFE_LEGEND explains every NF-e abbreviation, named like the filters', () => {
    const legend = NFE_LEGEND
    expect(badges(legend, 'Situação')).toEqual(['A', 'D', 'C'])
    expect(badges(legend, 'Completude')).toEqual(['R', 'X'])
    expect(badges(legend, 'Manifestação')).toEqual(['SM', 'CI', 'CO', 'DE', 'NR'])
    expect(badges(legend, 'Papel')).toEqual(['D', 'E', 'T', 'A', 'SP'])
    expect(names(legend, 'Situação')).toEqual(filterLabels(situacaoFilterOptions))
    expect(names(legend, 'Completude')).toEqual(filterLabels(completenessFilterOptions))
    expect(names(legend, 'Manifestação')).toEqual(filterLabels(manifestacaoFilterOptions))
    expect(names(legend, 'Papel')).toEqual(filterLabels(nfeRoleFilterOptions))
    expect(badges(legend, 'Prazo')).toEqual(['Confirmada tacitamente'])
  })

  it('NFE_LEGEND lists the groups in the order of the badges, then the deadline chip', () => {
    const titles = NFE_LEGEND.map((entry) => entry.title)
    const kinds = nfeStateBadges({
      Situacao: 'autorizada',
      Completeness: 'resumo',
      Manifestacao: 'nenhuma',
      CompanyRole: 'destinatario',
    }).map((badge) => badge.kind)
    expect(titles).toEqual(['Novo', ...kinds, 'Prazo'])
  })

  it('NFE_PENDING_LEGEND describes both pending groups with outlined chips', () => {
    const legend = NFE_PENDING_LEGEND
    expect(legend.map((entry) => entry.title)).toEqual(['Sem ciência', 'Sem manifestação conclusiva'])
    for (const entry of legend) {
      expect(entry.items.every((item) => item.outline)).toBe(true)
    }
    expect(badges(legend, 'Sem manifestação conclusiva')).toContain('Confirmada tacitamente')
  })

  it('NFE_PENDING_LEGEND colors the deadline chips as the table does', () => {
    const chips = (title: string) =>
      section(NFE_PENDING_LEGEND, title).items.map((item) => [item.badge, item.color])
    expect(chips('Sem ciência')).toEqual([
      ['10 d restantes', 'warning'],
      ['3 d restantes', 'negative'],
      ['Vence hoje', 'negative'],
      ['Vencido há 5 d', 'negative'],
    ])
    expect(chips('Sem manifestação conclusiva')).toEqual([
      ['45 d restantes', 'grey'],
      ['20 d restantes', 'warning'],
      ['5 d restantes', 'negative'],
      ['Confirmada tacitamente', 'negative'],
    ])
  })

  it('CTE_LEGEND explains every CT-e abbreviation, named by its label', () => {
    const legend = CTE_LEGEND
    expect(badges(legend, 'Documento')).toEqual(['CT', 'OS', 'GV', 'CS'])
    expect(badges(legend, 'Situação')).toEqual(['A', 'D', 'C'])
    expect(badges(legend, 'Papel')).toEqual(['TO', 'DE', 'RE', 'EX', 'RC', 'EM', 'AU', 'SP'])
    expect(names(legend, 'Papel')).toEqual(filterLabels(ctePapelFilterOptions))
    expect(names(legend, 'Situação')).toEqual(filterLabels(cteSituacaoFilterOptions))
    expect(names(legend, 'Documento')).toEqual(['CT-e', 'CT-e OS', 'GTV-e', 'CT-e Simplificado'])
  })

  it('CTE_LEGEND lists the groups in the order of the badges, without an export section', () => {
    const titles = CTE_LEGEND.map((entry) => entry.title)
    const kinds = cteStateBadges({
      Modelo: '57',
      TipoDocumento: 'cte',
      Situacao: 'autorizada',
      CompanyRole: 'tomador',
      Papeis: ['tomador'],
    }).map((badge) => badge.kind)
    expect(titles).toEqual(['Novo', ...kinds])
    expect(titles).not.toContain('Somente novos')
  })

  it('never repeats a badge inside a section', () => {
    for (const legend of [NFSE_LEGEND, NFE_LEGEND, NFE_PENDING_LEGEND, CTE_LEGEND]) {
      for (const entry of legend) {
        const seen = entry.items.map((item) => item.badge)
        expect(new Set(seen).size).toBe(seen.length)
      }
    }
  })
})

describe('NFSE_LEGEND', () => {
  it('explains every papel the Papel filter offers, with the table labels', () => {
    const papel = section(NFSE_LEGEND, 'Papel')
    expect(papel.items.map((entry) => entry.name)).toEqual([
      'Prestada',
      'Tomada',
      'Intermediário',
      'Sem papel fiscal',
    ])
    expect(papel.items[3]).toMatchObject({ badge: 'SP', color: 'grey' })
  })
})
