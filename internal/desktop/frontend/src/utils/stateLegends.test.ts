import { describe, expect, it } from 'vitest'
import {
  CTE_LEGEND,
  NFE_LEGEND,
  NFE_PENDING_LEGEND,
  NFSE_LEGEND,
  type LegendSection,
} from './stateLegends'
import {
  nfeCompleteness,
  nfeManifestacao,
  nfeRole,
  nfeSituacao,
  nfeStateBadges,
} from './nfeDisplay'
import { ctePapel, cteSituacao, cteStateBadges, cteTipoDocumento } from './cteDisplay'
import { nfseRole, nfseStateBadges, nfseStatus, nfseVisibility } from './nfseDisplay'
import type { DisplayTable } from './sefazDisplay'

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

// known lists the values of table a legend must explain: all of them, but
// the placeholder for an unknown value.
function known(table: DisplayTable) {
  return table.values().filter((value) => value !== 'unknown')
}

describe('stateLegends', () => {
  it('explains every known value of each table', () => {
    const cases: [LegendSection[], string, DisplayTable][] = [
      [NFSE_LEGEND, 'Status', nfseStatus],
      [NFSE_LEGEND, 'Visibilidade', nfseVisibility],
      [NFSE_LEGEND, 'Papel', nfseRole],
      [NFE_LEGEND, 'Situação', nfeSituacao],
      [NFE_LEGEND, 'Completude', nfeCompleteness],
      [NFE_LEGEND, 'Manifestação', nfeManifestacao],
      [NFE_LEGEND, 'Papel', nfeRole],
      [CTE_LEGEND, 'Documento', cteTipoDocumento],
      [CTE_LEGEND, 'Situação', cteSituacao],
      [CTE_LEGEND, 'Papel', ctePapel],
    ]
    for (const [legend, title, table] of cases) {
      expect(badges(legend, title), title).toEqual(known(table).map(table.abbr))
      expect(names(legend, title), title).toEqual(known(table).map(table.label))
    }
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

  it('NFE_LEGEND ends with the tacit confirmation chip', () => {
    expect(badges(NFE_LEGEND, 'Prazo')).toEqual(['Confirmada tacitamente'])
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
  it('lists the papéis in the order of the table and the Papel filter', () => {
    const papel = section(NFSE_LEGEND, 'Papel')
    expect(papel.items.map((entry) => entry.name)).toEqual([
      'Tomada',
      'Prestada',
      'Intermediário',
      'Sem papel fiscal',
    ])
    expect(papel.items[3]).toMatchObject({ badge: 'SP', color: 'grey' })
  })

  it('leaves the unknown visibilidade out', () => {
    expect(badges(NFSE_LEGEND, 'Visibilidade')).toEqual(['PE', 'TE', 'IE', 'MR'])
  })
})
