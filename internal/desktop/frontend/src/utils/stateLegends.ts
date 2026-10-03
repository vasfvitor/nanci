// Legends for the state badges of the document screens. Each page shows one
// StateLegend above its table. The entries come from the display tables, which
// hold the abbreviation, label, color and description of each value, so the
// legend never drifts from the badges.

import { nfseRole, nfseStatus, nfseVisibility } from './nfseDisplay'
import {
  DEADLINE_THRESHOLDS,
  deadlineColor,
  deadlineLabel,
  type DeadlineKind,
  nfeCompleteness,
  nfeManifestacao,
  nfeRole,
  nfeSituacao,
  TACIT_CONFIRMATION_LABEL,
} from './nfeDisplay'
import { ctePapel, cteSituacao, cteTipoDocumento } from './cteDisplay'
import { type LegendItem, type LegendSection, VIEWED_BADGE } from './sefazDisplay'

export type { LegendItem, LegendSection } from './sefazDisplay'

// deadlineItem builds the entry of a deadline chip with the label and color
// the table gives days of kind.
function deadlineItem(days: number, kind: DeadlineKind, description: string): LegendItem {
  return {
    badge: deadlineLabel(days),
    color: deadlineColor(days, kind),
    outline: true,
    description,
  }
}

// VIEWED_SECTION explains the "Novo" badge every document table shows.
const VIEWED_SECTION: LegendSection = {
  title: VIEWED_BADGE.label,
  items: [
    {
      badge: VIEWED_BADGE.label,
      color: VIEWED_BADGE.color,
      description:
        'Ninguém marcou este documento como visto. "Marcar vistos" tira a marca da seleção ou da lista exibida; "Somente não vistos" filtra por ela.',
    },
  ],
}

// NFSE_LEGEND follows the order of nfseStateBadges: status, visibilidade,
// papel.
export const NFSE_LEGEND: LegendSection[] = [
  VIEWED_SECTION,
  { title: 'Status', items: nfseStatus.legend() },
  {
    title: 'Visibilidade',
    note: 'Por que a nota chegou para a empresa.',
    items: nfseVisibility.legend(),
  },
  { title: 'Papel', items: nfseRole.legend() },
]

// NFE_LEGEND follows the order of nfeStateBadges: situação, completude,
// manifestação, papel; the deadline chip under the badges comes last.
export const NFE_LEGEND: LegendSection[] = [
  VIEWED_SECTION,
  { title: 'Situação', items: nfeSituacao.legend() },
  { title: 'Completude', items: nfeCompleteness.legend() },
  {
    title: 'Manifestação',
    note: 'Vem dos eventos que a própria empresa registrou. Uma manifestação conclusiva não se desfaz na SEFAZ.',
    items: nfeManifestacao.legend(),
  },
  { title: 'Papel', items: nfeRole.legend() },
  {
    title: 'Prazo',
    note: 'Nas notas com ciência, o chip sob as siglas conta os dias que faltam dos 90 para a manifestação conclusiva, a partir da autorização. Fica amarelo faltando 30 dias e vermelho faltando 10.',
    items: [
      {
        badge: TACIT_CONFIRMATION_LABEL,
        color: 'negative',
        outline: true,
        description:
          'Os 90 dias passaram sem manifestação conclusiva. Para a SEFAZ a operação ocorreu, como se tivesse sido confirmada.',
      },
    ],
  },
]

const ciencia = DEADLINE_THRESHOLDS.ciencia
const conclusiva = DEADLINE_THRESHOLDS.conclusiva

// NFE_PENDING_LEGEND explains the deadline chips of the Pendências tab. Their
// labels and colors come from deadlineLabel and deadlineColor, so they match
// the chips.
export const NFE_PENDING_LEGEND: LegendSection[] = [
  {
    title: 'Sem ciência',
    note: 'Notas de que a empresa é destinatária e ainda sem manifestação. O chip conta os dias que faltam dos 10 recomendados para a ciência, a partir da autorização.',
    items: [
      deadlineItem(10, 'ciencia', `Prazo perto do fim (${ciencia.warning} dias ou menos).`),
      deadlineItem(3, 'ciencia', `Últimos dias (${ciencia.urgent} dias ou menos).`),
      deadlineItem(0, 'ciencia', 'Último dia.'),
      deadlineItem(
        -5,
        'ciencia',
        'Ciência atrasada. Ainda pode ser registrada e continua liberando o XML completo.'
      ),
    ],
  },
  {
    title: 'Sem manifestação conclusiva',
    note: 'Notas com ciência e ainda sem manifestação conclusiva. O chip conta os dias que faltam dos 90, a partir da autorização.',
    items: [
      deadlineItem(45, 'conclusiva', 'Dentro do prazo.'),
      deadlineItem(20, 'conclusiva', `Prazo perto do fim (${conclusiva.warning} dias ou menos).`),
      deadlineItem(5, 'conclusiva', `Últimos dias (${conclusiva.urgent} dias ou menos).`),
      {
        badge: TACIT_CONFIRMATION_LABEL,
        color: 'negative',
        outline: true,
        description:
          'Os 90 dias passaram. Para a SEFAZ a operação ocorreu, como se tivesse sido confirmada, e a nota sai das pendências.',
      },
    ],
  },
]

// CTE_LEGEND follows the order of cteStateBadges: documento, situação,
// papel.
export const CTE_LEGEND: LegendSection[] = [
  VIEWED_SECTION,
  { title: 'Documento', items: cteTipoDocumento.legend() },
  { title: 'Situação', items: cteSituacao.legend() },
  {
    title: 'Papel',
    note: 'A empresa pode ter mais de um papel no mesmo CT-e. O principal vem na primeira linha; os outros, menores, na segunda.',
    items: ctePapel.legend(),
  },
]
