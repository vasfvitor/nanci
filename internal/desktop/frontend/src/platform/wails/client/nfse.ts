import {
  ExportDANFSe,
  ExportDocuments,
  ExportXML,
  ListDocuments,
  ListEventsForDocument,
  MarkDocumentsViewed,
  Pull,
  QueryNFSeEvents,
  ResetSyncState,
  TestConnection,
} from '../../../../wailsjs/go/main/App'
import type {
  ConnectionTestResult,
  DocumentEvent,
  DocumentRow,
  ExportDANFSeInput,
  ExportDocumentsInput,
  ExportFormat,
  ExportResult,
  ExportXMLInput,
  ListDocumentsInput,
  PullInput,
  PullResult,
  QueryNFSeInput,
  ResetSyncInput,
} from '@/types/desktop'
import {
  callWails,
  asString,
  asNumber,
  asNullableNumber,
  asRawRecord,
  asStringArray,
  asDate,
  type SaveDialog,
  fileTimestamp,
  mapExportResult,
  mapConnectionTestResult,
} from './base'
import { systemClient } from './system'

function nfseExportDialog(format: ExportFormat, cnpj: string): SaveDialog {
  if (format === 'danfse-zip') {
    return {
      title: 'Salvar ZIP de DANFSes',
      defaultName: `danfses_${cnpj}_${fileTimestamp()}.zip`,
      pattern: '*.zip',
    }
  }
  return {
    title: 'Exportar Documentos',
    defaultName: `nanci_exportacao_${cnpj}_${fileTimestamp()}.${format}`,
    pattern: `*.${format}`,
  }
}

export function mapDocumentRow(raw: unknown): DocumentRow {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    ChaveAcesso: asString(item['ChaveAcesso']),
    IssueDate: asDate(item['IssueDate']),
    Competence: asString(item['Competence']),
    PrestadorCNPJ: asString(item['PrestadorCNPJ']),
    PrestadorName: asString(item['PrestadorName']),
    TomadorCNPJ: asString(item['TomadorCNPJ']),
    TomadorName: asString(item['TomadorName']),
    IntermediarioCNPJ: asString(item['IntermediarioCNPJ']),
    IntermediarioName: asString(item['IntermediarioName']),
    ServiceValue: asNumber(item['ServiceValue']),
    ISSValue: asNumber(item['ISSValue']),
    IRRFValue: asNumber(item['IRRFValue']),
    INSSValue: asNumber(item['INSSValue']),
    PISValue: asNumber(item['PISValue']),
    COFINSValue: asNumber(item['COFINSValue']),
    CSLLValue: asNumber(item['CSLLValue']),
    TotalRetentions: asNumber(item['TotalRetentions']),
    Status: asString(item['Status']),
    LayoutVersion: asString(item['LayoutVersion']),
    XMLPath: asString(item['XMLPath']),
    RawHash: asString(item['RawHash']),
    ParseWarnings: asStringArray(item['ParseWarnings']),
    NFSeNumber: asString(item['NFSeNumber']),
    ServiceDescription: asString(item['ServiceDescription']),
    CreatedAt: asDate(item['CreatedAt']),
    UpdatedAt: asDate(item['UpdatedAt']),
    RelationID: asString(item['RelationID']),
    CompanyID: asString(item['CompanyID']),
    DocumentID: asString(item['DocumentID']),
    CompanyRole: asString(item['CompanyRole']),
    VisibilityReason: asString(item['VisibilityReason']),
    FirstSeenNSU: asNullableNumber(item['FirstSeenNSU']),
    LastSeenNSU: asNullableNumber(item['LastSeenNSU']),
    FirstSyncedAt: asDate(item['FirstSyncedAt']),
    LastSyncedAt: asDate(item['LastSyncedAt']),
    ViewedAt: asDate(item['ViewedAt']),
  }
}

export function mapDocumentEvent(raw: unknown): DocumentEvent {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    Type: asString(item['Type']),
    EventAt: item['EventAt'] as DocumentEvent['EventAt'],
    ReplacementChaveAcesso: asString(item['ReplacementChaveAcesso']),
    Description: asString(item['Description']),
    RawXMLPath: asString(item['RawXMLPath']),
  }
}

export const nfseClient = {
  async exportDocuments(
    input: Omit<ExportDocumentsInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const dialog = nfseExportDialog(input.Format, input.CNPJ)
    const defaultName = input.BaseName || dialog.defaultName
    const outPath =
      input.OutPath ||
      (await systemClient.selectSaveFile(dialog.title, defaultName, dialog.pattern))
    if (!outPath) return null

    const result = await callWails(() =>
      ExportDocuments({
        ...input,
        OutPath: outPath,
        ChavesAcesso: input.ChavesAcesso || [],
      })
    )
    return mapExportResult(result)
  },
  async exportDANFSe(
    input: Omit<ExportDANFSeInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `danfse_${input.ChaveAcesso}.pdf`
    const outPath =
      input.OutPath || (await systemClient.selectSaveFile('Salvar DANFSe', defaultName, '*.pdf'))
    if (!outPath) return null

    const result = await callWails(() =>
      ExportDANFSe({
        ...input,
        OutPath: outPath,
      })
    )
    return mapExportResult(result)
  },
  async exportXML(
    input: Omit<ExportXMLInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `nfse_${input.ChaveAcesso}.xml`
    const outPath =
      input.OutPath ||
      (await systemClient.selectSaveFile('Salvar XML Original', defaultName, '*.xml'))
    if (!outPath) return null

    const result = await callWails(() =>
      ExportXML({
        ...input,
        OutPath: outPath,
      })
    )
    return mapExportResult(result)
  },
  async listDocuments(input: ListDocumentsInput): Promise<DocumentRow[]> {
    const result = await callWails(() => ListDocuments(input))
    return (result || []).map(mapDocumentRow)
  },
  // markDocumentsViewed marks the company's NFS-e with these chaves and
  // returns how many were new.
  async markDocumentsViewed(cnpj: string, chavesAcesso: string[]): Promise<number> {
    return callWails(() => MarkDocumentsViewed({ CNPJ: cnpj, ChavesAcesso: chavesAcesso }))
  },
  async listEventsForDocument(documentID: string): Promise<DocumentEvent[]> {
    const result = await callWails(() => ListEventsForDocument(documentID))
    return (result || []).map(mapDocumentEvent)
  },
  pull(input: PullInput): Promise<PullResult> {
    return callWails(() => Pull(input)) as Promise<PullResult>
  },
  queryNFSeEvents(input: QueryNFSeInput) {
    return callWails(() => QueryNFSeEvents(input))
  },
  resetSyncState(input: ResetSyncInput) {
    return callWails(() => ResetSyncState(input))
  },
  async testConnection(companyCNPJ: string): Promise<ConnectionTestResult> {
    const res = await callWails(() => TestConnection(companyCNPJ))
    return mapConnectionTestResult(res)
  },
}
