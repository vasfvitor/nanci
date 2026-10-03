// Legends for the state badges of the document screens. Each page shows one
// StateLegend above its table; the abbreviations, labels and colors come from
// the display tables so the legend never drifts from the badges.

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
import { type DisplayTable, VIEWED_BADGE } from './sefazDisplay'

export type LegendItem = {
  // Text inside the badge, as the table shows it.
  badge: string
  color: string
  // Full name when the badge is an abbreviation.
  name?: string
  description: string
  // Rendered as an outlined chip instead of a filled badge.
  outline?: boolean
}

export type LegendSection = {
  title: string
  // Optional line under the title, for notes that apply to the whole group.
  note?: string
  items: LegendItem[]
}

// abbreviated builds the legend entry of one value of table: badge, color and
// name come from the table, the description from the caller.
function abbreviated(table: DisplayTable, value: string, description: string): LegendItem {
  return {
    badge: table.abbr(value),
    color: table.color(value),
    name: table.label(value),
    description,
  }
}

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
  {
    title: 'Status',
    items: [
      abbreviated(nfseStatus, 'normal', 'Nota válida.'),
      abbreviated(nfseStatus, 'cancelada', 'Nota cancelada por evento registrado no ADN.'),
      abbreviated(nfseStatus, 'substituida', 'Nota substituída por outra NFS-e.'),
    ],
  },
  {
    title: 'Visibilidade',
    note: 'Por que a nota chegou para a empresa.',
    items: [
      abbreviated(
        nfseVisibility,
        'exact_prestador',
        'A empresa é a prestadora da nota, pelo CNPJ exato.'
      ),
      abbreviated(
        nfseVisibility,
        'exact_tomador',
        'A empresa é a tomadora da nota, pelo CNPJ exato.'
      ),
      abbreviated(
        nfseVisibility,
        'exact_intermediario',
        'A empresa é a intermediária da nota, pelo CNPJ exato.'
      ),
      abbreviated(
        nfseVisibility,
        'same_root_only',
        'A nota é de outro estabelecimento da mesma raiz de CNPJ, matriz ou filial. A empresa não tem papel fiscal nela.'
      ),
    ],
  },
  {
    title: 'Papel',
    items: [
      abbreviated(nfseRole, 'prestada', 'A empresa prestou o serviço e emitiu a nota.'),
      abbreviated(
        nfseRole,
        'tomada',
        'A empresa tomou o serviço; a nota foi emitida por outro prestador.'
      ),
      abbreviated(nfseRole, 'intermediario', 'A empresa consta como intermediária do serviço.'),
      abbreviated(
        nfseRole,
        'none',
        'A nota chegou pela raiz do CNPJ; a empresa não é prestadora, tomadora nem intermediária.'
      ),
    ],
  },
]

// NFE_LEGEND follows the order of nfeStateBadges: situação, completude,
// manifestação, papel; the deadline chip under the badges comes last.
export const NFE_LEGEND: LegendSection[] = [
  VIEWED_SECTION,
  {
    title: 'Situação',
    items: [
      abbreviated(nfeSituacao, 'autorizada', 'Nota autorizada pela SEFAZ.'),
      abbreviated(
        nfeSituacao,
        'denegada',
        'A SEFAZ negou a autorização por irregularidade fiscal do emitente ou do destinatário. A nota não tem validade.'
      ),
      abbreviated(nfeSituacao, 'cancelada', 'Nota cancelada pelo emitente.'),
    ],
  },
  {
    title: 'Completude',
    items: [
      abbreviated(
        nfeCompleteness,
        'resumo',
        'Só o resumo chegou. A SEFAZ distribui o XML completo depois da Ciência da Operação ou de uma manifestação conclusiva.'
      ),
      abbreviated(
        nfeCompleteness,
        'completa',
        'O XML completo está guardado e pode ser exportado.'
      ),
    ],
  },
  {
    title: 'Manifestação',
    note: 'Vem dos eventos que a própria empresa registrou. Uma manifestação conclusiva não se desfaz na SEFAZ.',
    items: [
      abbreviated(
        nfeManifestacao,
        'nenhuma',
        'Nenhum evento de manifestação registrado pela empresa.'
      ),
      abbreviated(
        nfeManifestacao,
        'ciencia',
        'Ciência da Operação registrada. Libera o XML completo, mas não conclui nada: a nota ainda espera uma manifestação conclusiva.'
      ),
      abbreviated(
        nfeManifestacao,
        'confirmada',
        'Confirmação da Operação: a empresa atestou à SEFAZ que a operação ocorreu.'
      ),
      abbreviated(
        nfeManifestacao,
        'desconhecida',
        'Desconhecimento da Operação: a empresa declarou à SEFAZ que não reconhece a operação.'
      ),
      abbreviated(
        nfeManifestacao,
        'nao_realizada',
        'Operação não Realizada: a empresa declarou à SEFAZ que a operação não aconteceu, com justificativa.'
      ),
    ],
  },
  {
    title: 'Papel',
    items: [
      abbreviated(
        nfeRole,
        'destinatario',
        'A nota foi emitida contra o CNPJ da empresa. Só esse papel permite manifestar.'
      ),
      abbreviated(
        nfeRole,
        'emitente',
        'A empresa emitiu a nota. A SEFAZ não devolve ao emitente as próprias notas, então ela só aparece aqui se chegou por outro caminho.'
      ),
      abbreviated(nfeRole, 'transportador', 'A empresa é a transportadora da nota.'),
      abbreviated(
        nfeRole,
        'autorizado',
        'O CNPJ da empresa foi informado no grupo autXML, como autorizado a obter o XML.'
      ),
      abbreviated(
        nfeRole,
        'none',
        'A empresa só tem a mesma raiz de CNPJ de uma das partes, ou o motivo da distribuição não foi identificado.'
      ),
    ],
  },
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
  {
    title: 'Documento',
    items: [
      abbreviated(cteTipoDocumento, 'cte', 'Conhecimento de Transporte Eletrônico (modelo 57).'),
      abbreviated(
        cteTipoDocumento,
        'cte_os',
        'CT-e de Outros Serviços (modelo 67): transporte de pessoas, de valores ou excesso de bagagem.'
      ),
      abbreviated(
        cteTipoDocumento,
        'gtve',
        'Guia de Transporte de Valores Eletrônica (modelo 64).'
      ),
      abbreviated(
        cteTipoDocumento,
        'cte_simplificado',
        'CT-e Simplificado (modelo 57), com menos campos.'
      ),
    ],
  },
  {
    title: 'Situação',
    items: [
      abbreviated(cteSituacao, 'autorizada', 'CT-e autorizado pela SEFAZ.'),
      abbreviated(
        cteSituacao,
        'denegada',
        'A SEFAZ negou a autorização por irregularidade fiscal de uma das partes. O documento não tem validade.'
      ),
      abbreviated(cteSituacao, 'cancelada', 'CT-e cancelado pelo emitente.'),
    ],
  },
  {
    title: 'Papel',
    note: 'A empresa pode ter mais de um papel no mesmo CT-e. O principal vem na primeira linha; os outros, menores, na segunda.',
    items: [
      abbreviated(ctePapel, 'tomador', 'A empresa contratou o transporte e paga o frete.'),
      abbreviated(ctePapel, 'destinatario', 'A empresa é a destinatária da carga.'),
      abbreviated(ctePapel, 'remetente', 'A empresa é a remetente da carga.'),
      abbreviated(
        ctePapel,
        'expedidor',
        'A empresa entrega a carga ao transportador no lugar do remetente.'
      ),
      abbreviated(ctePapel, 'recebedor', 'A empresa recebe a carga no lugar do destinatário.'),
      abbreviated(
        ctePapel,
        'emitente',
        'A empresa emitiu o CT-e. A SEFAZ não devolve ao emitente os próprios documentos, então ele só aparece aqui se chegou por outro caminho.'
      ),
      abbreviated(
        ctePapel,
        'autorizado',
        'O CNPJ ou CPF da empresa foi informado no grupo autXML, como autorizado a obter o XML.'
      ),
      abbreviated(
        ctePapel,
        'none',
        'A empresa só tem a mesma raiz de CNPJ de uma das partes, ou o motivo da distribuição não foi identificado.'
      ),
    ],
  },
]
