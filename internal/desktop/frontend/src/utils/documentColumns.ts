import type { QTableColumn } from 'quasar'
import type { ISODateValue } from '@/types/desktop'
import { formatCents, formatDate } from '@/utils/formatters'

// The columns every document table shows, in order. Pages render the cells
// in their #body slot by these names.
export const DOCUMENT_COLUMN_NAMES = [
  'acoes',
  'issueDate',
  'numero',
  'chave',
  'emitente',
  'destinatario',
  'estados',
  'valor',
] as const

export type DocumentColumnName = (typeof DOCUMENT_COLUMN_NAMES)[number]

type DocumentRowBase = { ChaveAcesso: string; IssueDate?: ISODateValue }

// DocumentColumnSpec is what differs between the sources: the labels of the
// two party columns and where a row keeps its number, parties and value.
// emitente and destinatario return the text the column sorts by. numeroLabel
// defaults to "Nº / Série"; a source without série says only "Número".
export type DocumentColumnSpec<Row extends DocumentRowBase> = {
  emitenteLabel: string
  destinatarioLabel: string
  numeroLabel?: string
  numero: (row: Row) => string
  emitente: (row: Row) => string
  destinatario: (row: Row) => string
  valor: (row: Row) => number
}

// documentColumns builds the q-table columns of a document table: fixed names,
// order, alignment and sorting, with the source's labels and fields.
export function documentColumns<Row extends DocumentRowBase>(
  spec: DocumentColumnSpec<Row>
): QTableColumn<Row>[] {
  return [
    { name: 'acoes', label: 'Ações', field: () => '', align: 'center' },
    {
      name: 'issueDate',
      label: 'Emissão',
      field: (row) => row.IssueDate,
      sortable: true,
      align: 'left',
      classes: 'text-mono',
      format: (value: ISODateValue) => formatDate(value),
    },
    { name: 'numero', label: spec.numeroLabel ?? 'Nº / Série', field: spec.numero, align: 'left' },
    { name: 'chave', label: 'Chave de acesso', field: (row) => row.ChaveAcesso, align: 'left' },
    {
      name: 'emitente',
      label: spec.emitenteLabel,
      field: spec.emitente,
      sortable: true,
      align: 'left',
    },
    {
      name: 'destinatario',
      label: spec.destinatarioLabel,
      field: spec.destinatario,
      sortable: true,
      align: 'left',
    },
    { name: 'estados', label: 'Estados', field: () => '', align: 'center' },
    {
      name: 'valor',
      label: 'Valor (R$)',
      field: spec.valor,
      sortable: true,
      align: 'right',
      classes: 'text-mono',
      format: (value: number) => formatCents(value),
    },
  ]
}
