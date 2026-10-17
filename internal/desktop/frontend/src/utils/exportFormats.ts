// ExportFormatOption is one format the ExportDialog offers. resumos marks
// the formats that can take the NF-e resumos, so the dialog shows
// "Incluir resumos" only for them.
export type ExportFormatOption = { label: string; value: string; resumos?: boolean }

export const NFSE_EXPORT_FORMATS: ExportFormatOption[] = [
  { label: 'Planilha CSV', value: 'csv' },
  { label: 'Planilha Excel (XLSX)', value: 'xlsx' },
  { label: 'XMLs originais (ZIP)', value: 'zip' },
  { label: 'DANFSes (ZIP)', value: 'danfse-zip' },
]

export const NFE_EXPORT_FORMATS: ExportFormatOption[] = [
  { label: 'XMLs (ZIP)', value: 'zip', resumos: true },
]

export const CTE_EXPORT_FORMATS: ExportFormatOption[] = [{ label: 'XMLs (ZIP)', value: 'zip' }]

// exportedNoun names what an export in format writes, for the "N XMLs
// exportados" notice: 'DANFSe' for 'danfse-zip', 'XML' for 'zip',
// 'documento' for the spreadsheets.
export function exportedNoun(format: string): string {
  switch (format) {
    case 'danfse-zip':
      return 'DANFSe'
    case 'danfe-zip':
      return 'DANFE'
    case 'dacte-zip':
      return 'DACTE'
    case 'zip':
      return 'XML'
    default:
      return 'documento'
  }
}
