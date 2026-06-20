import {
  AddCompany,
  AddCredential,
  AssignCredentialToCompany,
  CancelCertPassword,
  CountPendingExports,
  ExportDANFSe,
  ExportDANFSeZIP,
  ExportDocuments,
  ExportLogs,
  ExportXML,
  GetBuildInfo,
  GetDataDirectory,
  ListCompanies,
  ListCredentials,
  ListDocuments,
  ListEventsForDocument,
  MarkDocumentsViewed,
  OpenDataDirectory,
  OpenLogsDirectory,
  Pull,
  QueryNFSeEvents,
  ResetSyncState,
  SelectCertificate,
  SelectExportDirectory,
  SelectSaveFile,
  SetLogLevel,
  SubmitCertPassword,
  TestConnection,
  UpdateCompany,
  UpdateCredentialData,
  UpdateCredentialPath,
} from '../../../wailsjs/go/main/App'
import type {
  AddCompanyInput,
  AddCredentialInput,
  AssignCredentialInput,
  BuildInfo,
  CompanySummary,
  ConnectionTestResult,
  CredentialSummary,
  DocumentEvent,
  DocumentRow,
  ExportDANFSeInput,
  ExportDocumentsInput,
  ExportResult,
  ExportXMLInput,
  ListDocumentsInput,
  PullInput,
  PullResult,
  QueryNFSeInput,
  ResetSyncInput,
  UpdateCompanyInput,
  UpdateCredentialDataInput,
  UpdateCredentialPathInput,
} from '@/types/desktop'

type RawRecord = Record<string, unknown>

export class WailsClientError extends Error {
  constructor(message: string, readonly cause?: unknown) {
    super(message)
    this.name = 'WailsClientError'
  }
}

function normalizeError(error: unknown): WailsClientError {
  if (error instanceof WailsClientError) return error
  if (error instanceof Error) return new WailsClientError(error.message, error)
  return new WailsClientError(String(error), error)
}

async function callWails<T>(operation: () => Promise<T>): Promise<T> {
  try {
    return await operation()
  } catch (error) {
    throw normalizeError(error)
  }
}

function asString(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function asNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function asBoolean(value: unknown) {
  return typeof value === 'boolean' ? value : false
}

function asRawRecord(value: unknown): RawRecord {
  return value && typeof value === 'object' ? (value as RawRecord) : {}
}

function asStringArray(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

export function mapCompanySummary(raw: unknown): CompanySummary {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    CNPJ: asString(item['CNPJ']),
    CNPJRoot: asString(item['CNPJRoot']),
    Name: asString(item['Name']),
    CredentialID: asString(item['CredentialID']),
    CredentialLabel: asString(item['CredentialLabel']),
    CredentialCertPath: asString(item['CredentialCertPath']),
    Environment: asString(item['Environment']),
    LastNSU: asNumber(item['LastNSU']),
    LastFoundNSU: asNumber(item['LastFoundNSU']),
    LastFoundNSUValid: asBoolean(item['LastFoundNSUValid']),
    LastSyncAt: item['LastSyncAt'] as CompanySummary['LastSyncAt'],
    LastRunStatus: asString(item['LastRunStatus']),
    LastRunStopReason: asString(item['LastRunStopReason']),
    CreatedAt: item['CreatedAt'] as CompanySummary['CreatedAt'],
    UpdatedAt: item['UpdatedAt'] as CompanySummary['UpdatedAt'],
  }
}

export function mapCredentialSummary(raw: unknown): CredentialSummary {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    Label: asString(item['Label']),
    CertPath: asString(item['CertPath']),
    OwnerCNPJ: asString(item['OwnerCNPJ']),
    OwnerCNPJRoot: asString(item['OwnerCNPJRoot']),
    FingerprintSHA256: asString(item['FingerprintSHA256']),
    SubjectName: asString(item['SubjectName']),
    NotBefore: item['NotBefore'] as CredentialSummary['NotBefore'],
    NotAfter: item['NotAfter'] as CredentialSummary['NotAfter'],
    InspectedAt: item['InspectedAt'] as CredentialSummary['InspectedAt'],
    CreatedAt: item['CreatedAt'] as CredentialSummary['CreatedAt'],
    UpdatedAt: item['UpdatedAt'] as CredentialSummary['UpdatedAt'],
  }
}

export function mapDocumentRow(raw: unknown): DocumentRow {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    ChaveAcesso: asString(item['ChaveAcesso']),
    IssueDate: item['IssueDate'] as DocumentRow['IssueDate'],
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
    CreatedAt: item['CreatedAt'] as DocumentRow['CreatedAt'],
    UpdatedAt: item['UpdatedAt'] as DocumentRow['UpdatedAt'],
    RelationID: asString(item['RelationID']),
    CompanyID: asString(item['CompanyID']),
    DocumentID: asString(item['DocumentID']),
    CompanyRole: asString(item['CompanyRole']),
    VisibilityReason: asString(item['VisibilityReason']),
    FirstSeenNSU: asNumber(item['FirstSeenNSU']),
    LastSeenNSU: asNumber(item['LastSeenNSU']),
    FirstSeenNSUValid: asBoolean(item['FirstSeenNSUValid']),
    LastSeenNSUValid: asBoolean(item['LastSeenNSUValid']),
    FirstSyncedAt: item['FirstSyncedAt'] as DocumentRow['FirstSyncedAt'],
    LastSyncedAt: item['LastSyncedAt'] as DocumentRow['LastSyncedAt'],
    ViewedAt: item['ViewedAt'] as DocumentRow['ViewedAt'],
  }
}

export function mapDocumentEvent(raw: unknown): DocumentEvent {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    Type: asString(item['Type']),
    EventAt: asString(item['EventAt']),
    ReplacementChaveAcesso: asString(item['ReplacementChaveAcesso']),
    Description: asString(item['Description']),
    RawXMLPath: asString(item['RawXMLPath']),
  }
}

export const desktopClient = {
  addCompany(input: AddCompanyInput) {
    return callWails(() => AddCompany(input))
  },
  addCredential(input: AddCredentialInput) {
    return callWails(() => AddCredential(input))
  },
  assignCredential(input: AssignCredentialInput) {
    return callWails(() => AssignCredentialToCompany(input))
  },
  async exportDocuments(input: Omit<ExportDocumentsInput, 'OutPath'> & { BaseName?: string; OutPath?: string }): Promise<ExportResult | null> {
    const extension = input.Format === 'csv' ? '.csv' : input.Format === 'xlsx' ? '.xlsx' : '.zip'
    const defaultName = input.BaseName || `export_${input.CNPJ}_${Date.now()}${extension}`
    const outPath = input.OutPath || await desktopClient.selectSaveFile('Exportar Documentos', defaultName, `*${extension}`)
    if (!outPath) return null

    const result = await callWails(() =>
      ExportDocuments({
        ...input,
        OutPath: outPath,
      })
    )
    return result as ExportResult
  },
  async exportDANFSe(input: Omit<ExportDANFSeInput, 'OutPath'> & { BaseName?: string; OutPath?: string }): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `danfse_${input.ChaveAcesso}.pdf`
    const outPath = input.OutPath || await desktopClient.selectSaveFile('Salvar DANFSe', defaultName, '*.pdf')
    if (!outPath) return null

    const result = await callWails(() =>
      ExportDANFSe({
        ...input,
        OutPath: outPath,
      })
    )
    return result as ExportResult
  },
  async exportXML(input: Omit<ExportXMLInput, 'OutPath'> & { BaseName?: string; OutPath?: string }): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `nfse_${input.ChaveAcesso}.xml`
    const outPath = input.OutPath || await desktopClient.selectSaveFile('Salvar XML Original', defaultName, '*.xml')
    if (!outPath) return null

    const result = await callWails(() =>
      ExportXML({
        ...input,
        OutPath: outPath,
      })
    )
    return result as ExportResult
  },
  async exportDANFSeZIP(input: Omit<ExportDocumentsInput, 'OutPath'> & { BaseName?: string; OutPath?: string }): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `danfses_${input.CNPJ}_${Date.now()}.zip`
    const outPath = input.OutPath || await desktopClient.selectSaveFile('Salvar ZIP de DANFSes', defaultName, '*.zip')
    if (!outPath) return null

    const result = await callWails(() =>
      ExportDANFSeZIP({
        ...input,
        OutPath: outPath,
      })
    )
    return result as ExportResult
  },
  async listCompanies(): Promise<CompanySummary[]> {
    const result = await callWails(() => ListCompanies())
    return (result || []).map(mapCompanySummary)
  },
  async listCredentials(): Promise<CredentialSummary[]> {
    const result = await callWails(() => ListCredentials())
    return (result || []).map(mapCredentialSummary)
  },
  async listDocuments(input: ListDocumentsInput): Promise<DocumentRow[]> {
    const result = await callWails(() => ListDocuments(input))
    return (result || []).map(mapDocumentRow)
  },
  async markDocumentsViewed(input: ListDocumentsInput): Promise<number> {
    return callWails(() => MarkDocumentsViewed(input))
  },
  async countPendingExports(input: ExportDocumentsInput): Promise<number> {
    return callWails(() => CountPendingExports(input))
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
  async selectCertificate(): Promise<string | null> {
    const path = await callWails(() => SelectCertificate())
    return path || null
  },
  async selectExportDirectory(): Promise<string | null> {
    const path = await callWails(() => SelectExportDirectory())
    return path || null
  },
  async selectSaveFile(title: string, defaultFilename: string, pattern: string): Promise<string | null> {
    const path = await callWails(() => SelectSaveFile(title, defaultFilename, pattern))
    return path || null
  },
  setLogLevel(level: string) {
    return callWails(() => SetLogLevel(level))
  },
  submitCertPassword(requestID: string, password: string) {
    return callWails(() => SubmitCertPassword(requestID, password))
  },
  cancelCertPassword(requestID: string) {
    return callWails(() => CancelCertPassword(requestID))
  },
  updateCompany(input: UpdateCompanyInput) {
    return callWails(() => UpdateCompany(input))
  },
  updateCredentialData(input: UpdateCredentialDataInput) {
    return callWails(() => UpdateCredentialData(input))
  },
  updateCredentialPath(input: UpdateCredentialPathInput) {
    return callWails(() => UpdateCredentialPath(input))
  },
  getBuildInfo(): Promise<BuildInfo> {
    return callWails(() => GetBuildInfo())
  },
  getDataDirectory(): Promise<string> {
    return callWails(() => GetDataDirectory())
  },
  openDataDirectory(): Promise<void> {
    return callWails(() => OpenDataDirectory())
  },
  openLogsDirectory(): Promise<void> {
    return callWails(() => OpenLogsDirectory())
  },
  async exportLogs(): Promise<string | null> {
    const path = await callWails(() => ExportLogs())
    return path || null
  },
  async testConnection(companyCNPJ: string): Promise<ConnectionTestResult> {
    const res = await callWails(() => TestConnection(companyCNPJ))
    const item = asRawRecord(res)
    return {
      certLoaded: asBoolean(item['certLoaded']),
      certSubject: asString(item['certSubject']),
      certExpiration: asString(item['certExpiration']),
      mtlsAccepted: asBoolean(item['mtlsAccepted']),
      endpointReached: asBoolean(item['endpointReached']),
      responseCode: asString(item['responseCode']),
      responseDetail: asString(item['responseDetail']),
      statusExplanation: asString(item['statusExplanation']),
    }
  },
}
