// Legends for the state badges of the document screens. Each page shows one
// StateLegend above its table; the abbreviations, labels and colors come from
// the display tables so the legend never drifts from the badges.

import { nfseRole, nfseStatus, nfseVisibility } from './nfseDisplay'
import {
  nfeCompleteness,
  nfeManifestacao,
  nfeRole,
  nfeSituacao,
  TACIT_CONFIRMATION_LABEL,
} from './nfeDisplay'
import { ctePapel, cteSituacao, cteTipoDocumento } from './cteDisplay'
import type { DisplayTable } from './sefazDisplay'

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

// abbreviated explains value of table as an abbreviated badge, named by its
// full label.
function abbreviated(table: DisplayTable, value: string, description: string): LegendItem {
  return {
    badge: table.abbr(value),
    color: table.color(value),
    name: table.label(value),
    description,
  }
}

function item(badge: string, color: string, description: string): LegendItem {
  return { badge, color, description }
}

// viewedSection explains the "Novo" badge every document table shows.
export function viewedSection(): LegendSection {
  return {
    title: 'Novo',
    items: [
      item(
        'Novo',
        'warning',
        'Documento ainda não visto. Sai de "Somente não vistos" quando você usa "Marcar vistos" na seleção ou na lista exibida.'
      ),
    ],
  }
}

// nfseLegend follows the order of nfseStateBadges: status, visibilidade,
// papel.
export function nfseLegend(): LegendSection[] {
  return [
    viewedSection(),
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
          'O CNPJ da empresa é exatamente o prestador da nota.'
        ),
        abbreviated(
          nfseVisibility,
          'exact_tomador',
          'O CNPJ da empresa é exatamente o tomador da nota.'
        ),
        abbreviated(
          nfseVisibility,
          'exact_intermediario',
          'O CNPJ da empresa é exatamente o intermediário da nota.'
        ),
        abbreviated(
          nfseVisibility,
          'same_root_only',
          'A nota é de outro estabelecimento com a mesma raiz de CNPJ (matriz ou filial). A empresa não tem papel fiscal nela.'
        ),
      ],
    },
    {
      title: 'Direção',
      items: [
        abbreviated(nfseRole, 'prestada', 'A empresa prestou o serviço e emitiu a nota.'),
        abbreviated(
          nfseRole,
          'tomada',
          'A empresa tomou o serviço; a nota foi emitida por outro prestador.'
        ),
        abbreviated(nfseRole, 'intermediario', 'A empresa consta como intermediária do serviço.'),
      ],
    },
  ]
}

// nfeLegend follows the order of nfeStateBadges: situação, completude,
// manifestação, papel; the deadline chip under the badges comes last.
export function nfeLegend(): LegendSection[] {
  return [
    viewedSection(),
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
          'Só os dados básicos chegaram. O XML completo é distribuído depois da Ciência da Operação ou de uma manifestação conclusiva.'
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
      note: 'Estado derivado dos eventos que a própria empresa registrou. As conclusivas são definitivas na SEFAZ.',
      items: [
        abbreviated(
          nfeManifestacao,
          'nenhuma',
          'Nenhum evento de manifestação registrado pela empresa.'
        ),
        abbreviated(
          nfeManifestacao,
          'ciencia',
          'Ciência da Operação registrada. Não é conclusiva: libera o XML completo e a nota ainda aguarda uma manifestação conclusiva.'
        ),
        abbreviated(
          nfeManifestacao,
          'confirmada',
          'Confirmação da Operação: a empresa confirma que a operação ocorreu.'
        ),
        abbreviated(
          nfeManifestacao,
          'desconhecida',
          'Desconhecimento da Operação: a empresa declara não reconhecer a operação.'
        ),
        abbreviated(
          nfeManifestacao,
          'nao_realizada',
          'Operação não Realizada: a operação foi solicitada mas não aconteceu, com justificativa.'
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
          'A empresa emitiu a nota. A SEFAZ não distribui as próprias notas; ela só aparece quando chega por outro motivo.'
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
          'A empresa só compartilha a raiz do CNPJ com alguma das partes, ou o motivo não foi identificado.'
        ),
      ],
    },
    {
      title: 'Prazo',
      note: 'Chip sob as siglas das notas com ciência: dias até o fim dos 90 dias para a manifestação conclusiva, contados da autorização. Amarelo a 30 dias do fim, vermelho a 10.',
      items: [
        {
          badge: TACIT_CONFIRMATION_LABEL,
          color: 'negative',
          outline: true,
          description:
            'Os 90 dias passaram sem manifestação conclusiva. A operação é considerada ocorrida, com os mesmos efeitos da confirmação.',
        },
      ],
    },
  ]
}

export function nfePendingLegend(): LegendSection[] {
  return [
    {
      title: 'Sem ciência',
      note: 'Notas em que a empresa é destinatária e ainda não registrou nenhuma manifestação. O chip mostra os dias até o fim dos 10 dias recomendados para a ciência, contados da autorização.',
      items: [
        { badge: '10 d restantes', color: 'grey', outline: true, description: 'Dentro do prazo.' },
        { badge: '3 d restantes', color: 'warning', outline: true, description: 'Prazo perto do fim (10 dias ou menos).' },
        { badge: 'Vence hoje', color: 'negative', outline: true, description: 'Último dia (3 dias ou menos).' },
        {
          badge: 'Vencido há 5 d',
          color: 'negative',
          outline: true,
          description: 'Ciência atrasada. Ela ainda pode ser registrada, e continua liberando o XML completo.',
        },
      ],
    },
    {
      title: 'Sem manifestação conclusiva',
      note: 'Notas com ciência registrada que ainda aguardam uma manifestação conclusiva. O chip mostra os dias até o fim dos 90 dias, contados da autorização.',
      items: [
        { badge: '45 d restantes', color: 'grey', outline: true, description: 'Dentro do prazo.' },
        { badge: '20 d restantes', color: 'warning', outline: true, description: 'Prazo perto do fim (30 dias ou menos).' },
        { badge: '5 d restantes', color: 'negative', outline: true, description: 'Últimos dias (10 dias ou menos).' },
        {
          badge: TACIT_CONFIRMATION_LABEL,
          color: 'negative',
          outline: true,
          description:
            'Os 90 dias passaram. A operação é considerada ocorrida, com os mesmos efeitos da confirmação, e a nota sai das pendências.',
        },
      ],
    },
  ]
}

// cteLegend follows the order of cteStateBadges: documento, situação,
// papel.
export function cteLegend(): LegendSection[] {
  return [
    viewedSection(),
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
        abbreviated(ctePapel, 'tomador', 'A empresa toma o serviço de transporte e escritura o frete.'),
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
          'A empresa emitiu o CT-e. A SEFAZ não distribui os próprios documentos; ele só aparece quando chega por outro motivo.'
        ),
        abbreviated(
          ctePapel,
          'autorizado',
          'O CNPJ ou CPF da empresa foi informado no grupo autXML, como autorizado a obter o XML.'
        ),
        abbreviated(
          ctePapel,
          'none',
          'A empresa só compartilha a raiz do CNPJ com alguma das partes, ou o motivo não foi identificado.'
        ),
      ],
    },
  ]
}
