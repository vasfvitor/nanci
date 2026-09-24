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
  ExportNFeXML,
  ExportNFeZIP,
  ExportXML,
  GetBuildInfo,
  GetDataDirectory,
  ListCompanies,
  ListCredentials,
  ListDocuments,
  ListEventsForDocument,
  ListNFe,
  ListNFeEvents,
  ListNFePendingManifestacoes,
  MarkDocumentsViewed,
  OpenDataDirectory,
  OpenLogsDirectory,
  PlanNFeCiencia,
  Pull,
  PullNFe,
  QueryNFSeEvents,
  RegisterNFeCiencia,
  RegisterNFeManifestacao,
  ResetNFe,
  ResetSyncState,
  SelectCertificate,
  SelectExportDirectory,
  SelectSaveFile,
  SetLogLevel,
  StatusNFe,
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
  ExportNFeXMLInput,
  ExportNFeZIPInput,
  ISODateValue,
  ListDocumentsInput,
  ListNFeInput,
  NFeBlockedReason,
  NFeCienciaPlan,
  NFeCompleteness,
  NFeEvent,
  NFeEventBatchResult,
  NFeEventOutcome,
  NFeEventResult,
  NFeExportResult,
  NFeManifestacao,
  NFePendingKind,
  NFePendingRow,
  NFeResetResult,
  NFeRole,
  NFeRow,
  NFeSituacao,
  NFeSkipped,
  NFeStatusResult,
  PullInput,
  PullNFeResult,
  PullResult,
  QueryNFSeInput,
  RegisterNFeManifestacaoInput,
  ResetSyncInput,
  UpdateCompanyInput,
  UpdateCredentialDataInput,
  UpdateCredentialPathInput,
} from '@/types/desktop'

type RawRecord = Record<string, unknown>

// WailsErrorCode names the backend errors the UI branches on. The desktop
// ErrorFormatter rejects bound-method promises with {code, message}.
export type WailsErrorCode = 'canceled' | 'sefaz_blocked' | 'sync_running' | ''

function asErrorCode(value: unknown): WailsErrorCode {
  return value === 'canceled' || value === 'sefaz_blocked' || value === 'sync_running' ? value : ''
}

export class WailsClientError extends Error {
  constructor(
    message: string,
    readonly code: WailsErrorCode = '',
    readonly cause?: unknown
  ) {
    super(message)
    this.name = 'WailsClientError'
  }
}

// wailsErrorCode reads the error code from any thrown value, so callers do not
// depend on the error having gone through the client.
export function wailsErrorCode(error: unknown): WailsErrorCode {
  return normalizeError(error).code
}

// errorMessage is the text to show for any thrown value.
export function errorMessage(error: unknown): string {
  return normalizeError(error).message
}

function normalizeError(error: unknown): WailsClientError {
  if (error instanceof WailsClientError) return error
  if (error instanceof Error) return new WailsClientError(error.message, '', error)
  if (error && typeof error === 'object' && 'message' in error) {
    const payload = asRawRecord(error)
    return new WailsClientError(asString(payload['message']), asErrorCode(payload['code']), error)
  }
  return new WailsClientError(String(error), '', error)
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

function asNullableNumber(value: unknown) {
  if (value === null || value === undefined) return null
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function asRawRecord(value: unknown): RawRecord {
  return value && typeof value === 'object' ? (value as RawRecord) : {}
}

function asStringArray(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function asDate(value: unknown): ISODateValue {
  if (typeof value === 'string' || value instanceof Date) return value
  return null
}

// asEnum keeps only known values. An unknown fiscal state becomes '' so the UI
// shows "Desconhecido" instead of guessing a real state.
function asEnum<T extends string>(value: unknown, allowed: readonly T[]): T | '' {
  return typeof value === 'string' && (allowed as readonly string[]).includes(value)
    ? (value as T)
    : ''
}

const nfeSituacoes: readonly NFeSituacao[] = ['autorizada', 'denegada', 'cancelada']
const nfeCompletenesses: readonly NFeCompleteness[] = ['resumo', 'completa']
const nfeManifestacoes: readonly NFeManifestacao[] = [
  'nenhuma',
  'ciencia',
  'confirmada',
  'desconhecida',
  'nao_realizada',
]
const nfeRoles: readonly NFeRole[] = ['destinatario', 'emitente', 'transportador', 'autorizado', 'none']
const nfeOutcomes: readonly NFeEventOutcome[] = ['registrada', 'ja_registrada', 'rejeitada', 'nao_enviada']
const nfePendingKinds: readonly NFePendingKind[] = ['sem_ciencia', 'sem_conclusiva']
const nfeBlockedReasons: readonly NFeBlockedReason[] = ['caught_up', 'consumo_indevido', 'rate_budget']

function fileTimestamp(now = new Date()) {
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  const h = String(now.getHours()).padStart(2, '0')
  const min = String(now.getMinutes()).padStart(2, '0')
  const s = String(now.getSeconds()).padStart(2, '0')
  return `${y}_${m}_${d}_${h}${min}${s}`
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
    UF: asString(item['UF']),
    LastFoundNSU: asNullableNumber(item['LastFoundNSU']),
    LastSyncAt: asDate(item['LastSyncAt']),
    SyncStartPolicy: asString(item['SyncStartPolicy']) as CompanySummary['SyncStartPolicy'],
    SyncStartDate: asDate(item['SyncStartDate']),
    InitialSyncDoneAt: asDate(item['InitialSyncDoneAt']),
    LastRunStatus: asString(item['LastRunStatus']),
    LastRunStopReason: asString(item['LastRunStopReason']),
    CreatedAt: asDate(item['CreatedAt']),
    UpdatedAt: asDate(item['UpdatedAt']),
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
    NotBefore: asDate(item['NotBefore']),
    NotAfter: asDate(item['NotAfter']),
    InspectedAt: asDate(item['InspectedAt']),
    CreatedAt: asDate(item['CreatedAt']),
    UpdatedAt: asDate(item['UpdatedAt']),
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

export function mapNFeRow(raw: unknown): NFeRow {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    DocumentID: asString(item['DocumentID']),
    ChaveAcesso: asString(item['ChaveAcesso']),
    Serie: asString(item['Serie']),
    Numero: asString(item['Numero']),
    IssueDate: asDate(item['IssueDate']),
    AuthorizedAt: asDate(item['AuthorizedAt']),
    Protocolo: asString(item['Protocolo']),
    TpNF: asString(item['TpNF']),
    EmitenteCNPJ: asString(item['EmitenteCNPJ']),
    EmitenteName: asString(item['EmitenteName']),
    EmitenteIE: asString(item['EmitenteIE']),
    DestinatarioCNPJ: asString(item['DestinatarioCNPJ']),
    DestinatarioName: asString(item['DestinatarioName']),
    TotalValue: asNumber(item['TotalValue']),
    Situacao: asEnum(item['Situacao'], nfeSituacoes),
    Completeness: asEnum(item['Completeness'], nfeCompletenesses),
    Manifestacao: asEnum(item['Manifestacao'], nfeManifestacoes),
    ManifestacaoAt: asDate(item['ManifestacaoAt']),
    CienciaDue: asDate(item['CienciaDue']),
    ConclusiveDue: asDate(item['ConclusiveDue']),
    CompanyRole: asEnum(item['CompanyRole'], nfeRoles),
    EventCount: asNumber(item['EventCount']),
    FirstSyncedAt: asDate(item['FirstSyncedAt']),
    LastSyncedAt: asDate(item['LastSyncedAt']),
    DaysLeft: asNullableNumber(item['DaysLeft']),
    CienciaDaysLeft: asNullableNumber(item['CienciaDaysLeft']),
    TacitlyConfirmed: asBoolean(item['TacitlyConfirmed']),
    CienciaBlockReason: asString(item['CienciaBlockReason']),
    ConclusiveBlockReason: asString(item['ConclusiveBlockReason']),
  }
}

export function mapNFePendingRow(raw: unknown): NFePendingRow {
  const item = asRawRecord(raw)
  return {
    ...mapNFeRow(item),
    Kind: asEnum(item['Kind'], nfePendingKinds),
    CienciaOverdue: asBoolean(item['CienciaOverdue']),
  }
}

export function mapNFeEvent(raw: unknown): NFeEvent {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    TpEvento: asString(item['TpEvento']),
    NSeqEvento: asNumber(item['NSeqEvento']),
    Description: asString(item['Description']),
    EventAt: asDate(item['EventAt']),
    RegisteredAt: asDate(item['RegisteredAt']),
    Protocolo: asString(item['Protocolo']),
    CStat: asString(item['CStat']),
    XMotivo: asString(item['XMotivo']),
    Justificativa: asString(item['Justificativa']),
    Correcao: asString(item['Correcao']),
    AutorCNPJ: asString(item['AutorCNPJ']),
    Completeness: asEnum(item['Completeness'], nfeCompletenesses),
    Registered: asBoolean(item['Registered']),
    SentByNanci: asBoolean(item['SentByNanci']),
  }
}

export function mapNFeEventResult(raw: unknown): NFeEventResult {
  const item = asRawRecord(raw)
  return {
    ChaveAcesso: asString(item['ChaveAcesso']),
    TpEvento: asString(item['TpEvento']),
    Status: asEnum(item['Status'], nfeOutcomes),
    CStat: asString(item['CStat']),
    XMotivo: asString(item['XMotivo']),
    Protocolo: asString(item['Protocolo']),
    RegisteredAt: asDate(item['RegisteredAt']),
  }
}

function mapNFeSkipped(raw: unknown): NFeSkipped {
  const item = asRawRecord(raw)
  return {
    ChaveAcesso: asString(item['ChaveAcesso']),
    Reason: asString(item['Reason']),
  }
}

export function mapNFeEventBatchResult(raw: unknown): NFeEventBatchResult {
  const item = asRawRecord(raw)
  return {
    Results: asArray(item['Results']).map(mapNFeEventResult),
    Skipped: asArray(item['Skipped']).map(mapNFeSkipped),
    Interrupted: asString(item['Interrupted']),
  }
}

export function mapNFeCienciaPlan(raw: unknown): NFeCienciaPlan {
  const item = asRawRecord(raw)
  return {
    Eligible: asArray(item['Eligible']).map(mapNFeRow),
    Skipped: asArray(item['Skipped']).map(mapNFeSkipped),
  }
}

export function mapNFeStatus(raw: unknown): NFeStatusResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    UF: asString(item['UF']),
    TpAmb: asString(item['TpAmb']),
    LastNSU: asNumber(item['LastNSU']),
    MaxNSU: asNullableNumber(item['MaxNSU']),
    LastSyncAt: asDate(item['LastSyncAt']),
    LastRunStatus: asString(item['LastRunStatus']),
    LastRunStopReason: asString(item['LastRunStopReason']),
    InitialSyncDoneAt: asDate(item['InitialSyncDoneAt']),
    NextAllowedAt: asDate(item['NextAllowedAt']),
    BlockedReason: asEnum(item['BlockedReason'], nfeBlockedReasons),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
    TotalDestinatario: asNumber(item['TotalDestinatario']),
    TotalEmitente: asNumber(item['TotalEmitente']),
    TotalOutros: asNumber(item['TotalOutros']),
    TotalResumos: asNumber(item['TotalResumos']),
    TotalCompletas: asNumber(item['TotalCompletas']),
    PendingCiencia: asNumber(item['PendingCiencia']),
    PendingConclusiva: asNumber(item['PendingConclusiva']),
    CienciaOverdue: asNumber(item['CienciaOverdue']),
  }
}

export function mapPullNFeResult(raw: unknown): PullNFeResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    Status: asString(item['Status']),
    StopReason: asString(item['StopReason']),
    LastNSU: asNumber(item['LastNSU']),
    MaxNSU: asNullableNumber(item['MaxNSU']),
    CompletasSaved: asNumber(item['CompletasSaved']),
    ResumosSaved: asNumber(item['ResumosSaved']),
    EventsSaved: asNumber(item['EventsSaved']),
    Errors: asNumber(item['Errors']),
    NextAllowedAt: asDate(item['NextAllowedAt']),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
    Duration: asNumber(item['Duration']),
  }
}

function mapExportResult(raw: unknown): ExportResult {
  const item = asRawRecord(raw)
  return {
    OutPath: asString(item['OutPath']),
    Format: asString(item['Format']) as ExportResult['Format'],
    Incremental: asBoolean(item['Incremental']),
    ExportedCount: asNumber(item['ExportedCount']),
  }
}

function mapNFeExportResult(raw: unknown): NFeExportResult {
  return {
    ...mapExportResult(raw),
    SkippedResumos: asNumber(asRawRecord(raw)['SkippedResumos']),
  }
}

export function mapNFeResetResult(raw: unknown): NFeResetResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    CompanyDocuments: asNumber(item['CompanyDocuments']),
    Documents: asNumber(item['Documents']),
    Events: asNumber(item['Events']),
    ExportMarks: asNumber(item['ExportMarks']),
    ManifestacoesKept: asNumber(item['ManifestacoesKept']),
  }
}

function mapConnectionTestResult(raw: unknown): ConnectionTestResult {
  const item = asRawRecord(raw)
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
    const defaultName = input.BaseName || `nanci_exportacao_${input.CNPJ}_${fileTimestamp()}${extension}`
    const outPath = input.OutPath || await desktopClient.selectSaveFile('Exportar Documentos', defaultName, `*${extension}`)
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
    return mapExportResult(result)
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
    return mapExportResult(result)
  },
  async exportDANFSeZIP(input: Omit<ExportDocumentsInput, 'OutPath'> & { BaseName?: string; OutPath?: string }): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `danfses_${input.CNPJ}_${Date.now()}.zip`
    const outPath = input.OutPath || await desktopClient.selectSaveFile('Salvar ZIP de DANFSes', defaultName, '*.zip')
    if (!outPath) return null

    const result = await callWails(() =>
      ExportDANFSeZIP({
        ...input,
        OutPath: outPath,
        ChavesAcesso: input.ChavesAcesso || [],
      })
    )
    return mapExportResult(result)
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
    return mapConnectionTestResult(res)
  },

  // NF-e (modelo 55)
  async pullNFe(cnpj: string): Promise<PullNFeResult> {
    const res = await callWails(() => PullNFe({ CNPJ: cnpj }))
    return mapPullNFeResult(res)
  },
  // resetNFe removes the company's NF-e and resets its NF-e sync.
  async resetNFe(cnpj: string): Promise<NFeResetResult> {
    const res = await callWails(() => ResetNFe(cnpj))
    return mapNFeResetResult(res)
  },
  async statusNFe(cnpj: string): Promise<NFeStatusResult> {
    const res = await callWails(() => StatusNFe(cnpj))
    return mapNFeStatus(res)
  },
  async listNFe(input: ListNFeInput): Promise<NFeRow[]> {
    const res = await callWails(() => ListNFe({ ...input, ChavesAcesso: input.ChavesAcesso ?? [] }))
    return (res || []).map(mapNFeRow)
  },
  async listNFeEvents(cnpj: string, chaveAcesso: string): Promise<NFeEvent[]> {
    const res = await callWails(() => ListNFeEvents({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
    return (res || []).map(mapNFeEvent)
  },
  // dueWithinDays 0 lists every pending manifestação.
  async listNFePendingManifestacoes(cnpj: string, dueWithinDays = 0): Promise<NFePendingRow[]> {
    const res = await callWails(() =>
      ListNFePendingManifestacoes({ CNPJ: cnpj, DueWithinDays: dueWithinDays })
    )
    return (res || []).map(mapNFePendingRow)
  },
  async planNFeCiencia(cnpj: string, chavesAcesso: string[]): Promise<NFeCienciaPlan> {
    const res = await callWails(() => PlanNFeCiencia({ CNPJ: cnpj, ChavesAcesso: chavesAcesso }))
    return mapNFeCienciaPlan(res)
  },
  async registerNFeCiencia(cnpj: string, chavesAcesso: string[]): Promise<NFeEventBatchResult> {
    const res = await callWails(() => RegisterNFeCiencia({ CNPJ: cnpj, ChavesAcesso: chavesAcesso }))
    return mapNFeEventBatchResult(res)
  },
  async registerNFeManifestacao(input: RegisterNFeManifestacaoInput): Promise<NFeEventResult> {
    const res = await callWails(() => RegisterNFeManifestacao(input))
    return mapNFeEventResult(res)
  },
  async exportNFeXML(
    input: Omit<ExportNFeXMLInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `nfe_${input.ChaveAcesso}.xml`
    const outPath =
      input.OutPath || (await desktopClient.selectSaveFile('Salvar XML da NF-e', defaultName, '*.xml'))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportNFeXML({ CNPJ: input.CNPJ, ChaveAcesso: input.ChaveAcesso, OutPath: outPath })
    )
    return mapExportResult(res)
  },
  async exportNFeZIP(
    input: Omit<ExportNFeZIPInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<NFeExportResult | null> {
    const defaultName = input.BaseName || `nfe_${input.CNPJ}_${fileTimestamp()}.zip`
    const outPath =
      input.OutPath ||
      (await desktopClient.selectSaveFile('Salvar XMLs de NF-e (ZIP)', defaultName, '*.zip'))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportNFeZIP({
        CNPJ: input.CNPJ,
        Competence: input.Competence,
        Role: input.Role,
        ChavesAcesso: input.ChavesAcesso || [],
        IncludeResumos: input.IncludeResumos,
        Incremental: input.Incremental,
        OutPath: outPath,
      })
    )
    return mapNFeExportResult(res)
  },
}
