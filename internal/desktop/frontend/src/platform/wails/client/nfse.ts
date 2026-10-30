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
} from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'
import type * as desktopapi from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'
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
  asNullableNumber,
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

export function mapDocumentRow(item: desktopapi.DocumentRow): DocumentRow {
  return {
    ID: item.ID,
    ChaveAcesso: item.ChaveAcesso,
    IssueDate: item.IssueDate,
    Competence: item.Competence,
    PrestadorCNPJ: item.PrestadorCNPJ,
    PrestadorName: item.PrestadorName,
    TomadorCNPJ: item.TomadorCNPJ,
    TomadorName: item.TomadorName,
    IntermediarioCNPJ: item.IntermediarioCNPJ,
    IntermediarioName: item.IntermediarioName,
    ServiceValue: item.ServiceValue,
    ISSValue: item.ISSValue,
    IRRFValue: item.IRRFValue,
    INSSValue: item.INSSValue,
    PISValue: item.PISValue,
    COFINSValue: item.COFINSValue,
    CSLLValue: item.CSLLValue,
    TotalRetentions: item.TotalRetentions,
    Status: item.Status,
    LayoutVersion: item.LayoutVersion,
    XMLPath: item.XMLPath,
    RawHash: item.RawHash,
    ParseWarnings: asStringArray(item.ParseWarnings),
    NFSeNumber: item.NFSeNumber,
    ServiceDescription: item.ServiceDescription,
    CreatedAt: item.CreatedAt,
    UpdatedAt: item.UpdatedAt,
    RelationID: item.RelationID,
    CompanyID: item.CompanyID,
    DocumentID: item.DocumentID,
    CompanyRole: item.CompanyRole,
    VisibilityReason: item.VisibilityReason,
    FirstSeenNSU: asNullableNumber(item.FirstSeenNSU),
    LastSeenNSU: asNullableNumber(item.LastSeenNSU),
    FirstSyncedAt: item.FirstSyncedAt,
    LastSyncedAt: item.LastSyncedAt,
    ViewedAt: asDate(item.ViewedAt),
  }
}

export function mapDocumentEvent(item: desktopapi.DocumentEvent): DocumentEvent {
  return {
    ID: item.ID,
    Type: item.Type,
    EventAt: item.EventAt ?? null,
    ReplacementChaveAcesso: item.ReplacementChaveAcesso,
    Description: item.Description,
    RawXMLPath: item.RawXMLPath,
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
