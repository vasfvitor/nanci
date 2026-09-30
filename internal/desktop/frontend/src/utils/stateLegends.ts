// Legends for the state badges of the document screens. Each page shows one
// StateLegend above its table; the labels and colors come from the display
// tables so the legend never drifts from the badges.

import {
  getRoleAbbreviation,
  getStatusAbbreviation,
  getStatusLabel,
  getVisibilityAbbreviation,
  roleColor,
  roleLabel,
  statusColor,
  visibilityColor,
  visibilityLabel,
} from './nfseDisplay'
import {
  completenessColor,
  completenessLabel,
  manifestacaoColor,
  manifestacaoLabel,
  nfeRoleColor,
  nfeRoleLabel,
  situacaoColor,
  situacaoLabel,
  TACIT_CONFIRMATION_LABEL,
} from './nfeDisplay'
import {
  ctePapelColor,
  ctePapelLabel,
  cteSituacaoColor,
  cteSituacaoLabel,
  cteTipoDocumentoColor,
  cteTipoDocumentoLabel,
} from './cteDisplay'

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

function abbreviated(badge: string, color: string, name: string, description: string): LegendItem {
  return { badge, color, name, description }
}

function item(badge: string, color: string, description: string): LegendItem {
  return { badge, color, description }
}

export function nfseLegend(): LegendSection[] {
  return [
    {
      title: 'Novo',
      items: [
        item(
          'Novo',
          'warning',
          'Documento ainda não marcado como visto. Ele só sai de "Somente novos" quando você clica em "Marcar Vistos", que marca os documentos da competência, direção e status filtrados. Exportar ou abrir os detalhes não marca.'
        ),
      ],
    },
    {
      title: 'Direção',
      items: [
        abbreviated(
          getRoleAbbreviation('prestada'),
          roleColor('prestada'),
          roleLabel('prestada'),
          'A empresa prestou o serviço e emitiu a nota.'
        ),
        abbreviated(
          getRoleAbbreviation('tomada'),
          roleColor('tomada'),
          roleLabel('tomada'),
          'A empresa tomou o serviço; a nota foi emitida por outro prestador.'
        ),
        abbreviated(
          getRoleAbbreviation('intermediario'),
          roleColor('intermediario'),
          roleLabel('intermediario'),
          'A empresa consta como intermediária do serviço.'
        ),
      ],
    },
    {
      title: 'Visibilidade',
      note: 'Por que a nota chegou para a empresa.',
      items: [
        abbreviated(
          getVisibilityAbbreviation('exact_prestador'),
          visibilityColor('exact_prestador'),
          visibilityLabel('exact_prestador'),
          'O CNPJ da empresa é exatamente o prestador da nota.'
        ),
        abbreviated(
          getVisibilityAbbreviation('exact_tomador'),
          visibilityColor('exact_tomador'),
          visibilityLabel('exact_tomador'),
          'O CNPJ da empresa é exatamente o tomador da nota.'
        ),
        abbreviated(
          getVisibilityAbbreviation('exact_intermediario'),
          visibilityColor('exact_intermediario'),
          visibilityLabel('exact_intermediario'),
          'O CNPJ da empresa é exatamente o intermediário da nota.'
        ),
        abbreviated(
          getVisibilityAbbreviation('same_root_only'),
          visibilityColor('same_root_only'),
          visibilityLabel('same_root_only'),
          'A nota é de outro estabelecimento com a mesma raiz de CNPJ (matriz ou filial). A empresa não tem papel fiscal nela.'
        ),
      ],
    },
    {
      title: 'Status',
      items: [
        abbreviated(
          getStatusAbbreviation('normal'),
          statusColor('normal'),
          getStatusLabel('normal'),
          'Nota válida.'
        ),
        abbreviated(
          getStatusAbbreviation('cancelada'),
          statusColor('cancelada'),
          getStatusLabel('cancelada'),
          'Nota cancelada por evento registrado no ADN.'
        ),
        abbreviated(
          getStatusAbbreviation('substituida'),
          statusColor('substituida'),
          getStatusLabel('substituida'),
          'Nota substituída por outra NFS-e.'
        ),
      ],
    },
  ]
}

function nfeSituacaoSection(): LegendSection {
  return {
    title: 'Situação',
    items: [
      item(situacaoLabel('autorizada'), situacaoColor('autorizada'), 'Nota autorizada pela SEFAZ.'),
      item(
        situacaoLabel('denegada'),
        situacaoColor('denegada'),
        'A SEFAZ negou a autorização por irregularidade fiscal do emitente ou do destinatário. A nota não tem validade.'
      ),
      item(situacaoLabel('cancelada'), situacaoColor('cancelada'), 'Nota cancelada pelo emitente.'),
    ],
  }
}

export function nfeLegend(): LegendSection[] {
  return [
    nfeSituacaoSection(),
    {
      title: 'Completude',
      items: [
        item(
          completenessLabel('resumo'),
          completenessColor('resumo'),
          'Só os dados básicos chegaram. O XML completo é distribuído depois da Ciência da Operação ou de uma manifestação conclusiva.'
        ),
        item(
          completenessLabel('completa'),
          completenessColor('completa'),
          'O XML completo está guardado e pode ser exportado.'
        ),
      ],
    },
    {
      title: 'Manifestação',
      note: 'Estado derivado dos eventos que a própria empresa registrou. As conclusivas são definitivas na SEFAZ.',
      items: [
        item(
          manifestacaoLabel('nenhuma'),
          manifestacaoColor('nenhuma'),
          'Nenhum evento de manifestação registrado pela empresa.'
        ),
        item(
          manifestacaoLabel('ciencia'),
          manifestacaoColor('ciencia'),
          'Ciência da Operação registrada. Não é conclusiva: libera o XML completo e a nota ainda aguarda uma manifestação conclusiva.'
        ),
        item(
          manifestacaoLabel('confirmada'),
          manifestacaoColor('confirmada'),
          'Confirmação da Operação: a empresa confirma que a operação ocorreu.'
        ),
        item(
          manifestacaoLabel('desconhecida'),
          manifestacaoColor('desconhecida'),
          'Desconhecimento da Operação: a empresa declara não reconhecer a operação.'
        ),
        item(
          manifestacaoLabel('nao_realizada'),
          manifestacaoColor('nao_realizada'),
          'Operação não Realizada: a operação foi solicitada mas não aconteceu, com justificativa.'
        ),
      ],
    },
    {
      title: 'Prazo',
      note: 'Chip ao lado da ciência: dias até o fim dos 90 dias para a manifestação conclusiva, contados da autorização. Amarelo a 30 dias do fim, vermelho a 10.',
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
    {
      title: 'Papel',
      items: [
        item(
          nfeRoleLabel('destinatario'),
          nfeRoleColor('destinatario'),
          'A nota foi emitida contra o CNPJ da empresa. Só esse papel permite manifestar.'
        ),
        item(
          nfeRoleLabel('emitente'),
          nfeRoleColor('emitente'),
          'A empresa emitiu a nota. A SEFAZ não distribui as próprias notas; ela só aparece quando chega por outro motivo.'
        ),
        item(nfeRoleLabel('transportador'), nfeRoleColor('transportador'), 'A empresa é a transportadora da nota.'),
        item(
          nfeRoleLabel('autorizado'),
          nfeRoleColor('autorizado'),
          'O CNPJ da empresa foi informado no grupo autXML, como autorizado a obter o XML.'
        ),
        item(
          nfeRoleLabel('none'),
          nfeRoleColor('none'),
          'A empresa só compartilha a raiz do CNPJ com alguma das partes, ou o motivo não foi identificado.'
        ),
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

export function cteLegend(): LegendSection[] {
  return [
    {
      title: 'Documento',
      items: [
        item(cteTipoDocumentoLabel('cte'), cteTipoDocumentoColor('cte'), 'Conhecimento de Transporte Eletrônico (modelo 57).'),
        item(
          cteTipoDocumentoLabel('cte_os'),
          cteTipoDocumentoColor('cte_os'),
          'CT-e de Outros Serviços (modelo 67): transporte de pessoas, de valores ou excesso de bagagem.'
        ),
        item(cteTipoDocumentoLabel('gtve'), cteTipoDocumentoColor('gtve'), 'Guia de Transporte de Valores Eletrônica (modelo 64).'),
        item(
          cteTipoDocumentoLabel('cte_simplificado'),
          cteTipoDocumentoColor('cte_simplificado'),
          'CT-e Simplificado (modelo 57), com menos campos.'
        ),
      ],
    },
    {
      title: 'Papel',
      note: 'A empresa pode ter mais de um papel no mesmo CT-e. O principal vem em cima; os outros, menores, embaixo.',
      items: [
        item(
          ctePapelLabel('tomador'),
          ctePapelColor('tomador'),
          'A empresa toma o serviço de transporte e escritura o frete.'
        ),
        item(ctePapelLabel('destinatario'), ctePapelColor('destinatario'), 'A empresa é a destinatária da carga.'),
        item(ctePapelLabel('remetente'), ctePapelColor('remetente'), 'A empresa é a remetente da carga.'),
        item(
          ctePapelLabel('expedidor'),
          ctePapelColor('expedidor'),
          'A empresa entrega a carga ao transportador no lugar do remetente.'
        ),
        item(ctePapelLabel('recebedor'), ctePapelColor('recebedor'), 'A empresa recebe a carga no lugar do destinatário.'),
        item(
          ctePapelLabel('emitente'),
          ctePapelColor('emitente'),
          'A empresa emitiu o CT-e. A SEFAZ não distribui os próprios documentos; ele só aparece quando chega por outro motivo.'
        ),
        item(
          ctePapelLabel('autorizado'),
          ctePapelColor('autorizado'),
          'O CNPJ ou CPF da empresa foi informado no grupo autXML, como autorizado a obter o XML.'
        ),
        item(
          ctePapelLabel('none'),
          ctePapelColor('none'),
          'A empresa só compartilha a raiz do CNPJ com alguma das partes, ou o motivo não foi identificado.'
        ),
      ],
    },
    {
      title: 'Situação',
      items: [
        item(cteSituacaoLabel('autorizada'), cteSituacaoColor('autorizada'), 'CT-e autorizado pela SEFAZ.'),
        item(
          cteSituacaoLabel('denegada'),
          cteSituacaoColor('denegada'),
          'A SEFAZ negou a autorização por irregularidade fiscal de uma das partes. O documento não tem validade.'
        ),
        item(cteSituacaoLabel('cancelada'), cteSituacaoColor('cancelada'), 'CT-e cancelado pelo emitente.'),
      ],
    },
    {
      title: 'Somente novos',
      note: 'Opção da exportação, não da lista: o ZIP leva só os CT-e ainda não exportados ou cujo XML mudou desde a última exportação. Um CT-e volta a contar como novo quando recebe um evento.',
      items: [],
    },
  ]
}
