export type ISODateValue = string | Date | null | undefined

export type CompanySummary = {
  ID: string
  CNPJ: string
  CNPJRoot: string
  Name: string
  CredentialID: string
  CredentialLabel: string
  CredentialCertPath: string
  Environment: string
  UF: string
  LastFoundNSU: number | null
  LastSyncAt?: ISODateValue
  SyncStartPolicy: SyncStartPolicy
  SyncStartDate?: ISODateValue
  InitialSyncDoneAt?: ISODateValue
  LastRunStatus: string
  LastRunStopReason: string
  CreatedAt?: ISODateValue
  UpdatedAt?: ISODateValue
}

export type SyncStartPolicy = 'all' | 'since_date' | 'from_now' | ''

export type CredentialSummary = {
  ID: string
  Label: string
  CertPath: string
  OwnerCNPJ: string
  OwnerCNPJRoot: string
  FingerprintSHA256: string
  SubjectName: string
  NotBefore?: ISODateValue
  NotAfter?: ISODateValue
  InspectedAt?: ISODateValue
  CreatedAt?: ISODateValue
  UpdatedAt?: ISODateValue
}

export type DocumentRow = {
  ID: string
  ChaveAcesso: string
  IssueDate?: ISODateValue
  Competence: string
  PrestadorCNPJ: string
  PrestadorName: string
  TomadorCNPJ: string
  TomadorName: string
  IntermediarioCNPJ: string
  IntermediarioName: string
  ServiceValue: number
  ISSValue: number
  IRRFValue: number
  INSSValue: number
  PISValue: number
  COFINSValue: number
  CSLLValue: number
  TotalRetentions: number
  Status: string
  LayoutVersion: string
  XMLPath: string
  RawHash: string
  ParseWarnings: string[]
  NFSeNumber: string
  ServiceDescription: string
  CreatedAt?: ISODateValue
  UpdatedAt?: ISODateValue
  RelationID: string
  CompanyID: string
  DocumentID: string
  CompanyRole: string
  VisibilityReason: string
  FirstSeenNSU: number | null
  LastSeenNSU: number | null
  FirstSyncedAt?: ISODateValue
  LastSyncedAt?: ISODateValue
  ViewedAt?: ISODateValue
}

export type DocumentEvent = {
  ID: string
  Type: string
  EventAt: string | null
  ReplacementChaveAcesso: string
  Description: string
  RawXMLPath: string
}

export type ListDocumentsInput = {
  CNPJ: string
  Competence: string
  Direction: string
  OnlyUnread: boolean
}

export type ExportFormat = 'csv' | 'xlsx' | 'zip'
export type ExportResultFormat = ExportFormat | 'danfse' | 'danfse-zip'

export type ExportDocumentsInput = {
  CNPJ: string
  Competence: string
  Direction: string
  Format: ExportFormat
  OutPath: string
  Incremental: boolean
  ChavesAcesso: string[]
}

export type ExportDANFSeInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

export type ExportResult = {
  OutPath: string
  Format: ExportResultFormat
  Incremental: boolean
  ExportedCount: number
}

export type QueryNFSeInput = {
  CompanyCNPJ: string
  ChaveAcesso: string
}

export type ExportXMLInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

export type AddCompanyInput = {
  CNPJ: string
  Name: string
  CredentialID: string
  CredentialLabel: string
  CertPath: string
  Environment: string
  UF: string
  SyncStartPolicy: SyncStartPolicy
  SyncStartDate: string
}

export type AddCredentialInput = {
  Label: string
  CertPath: string
}

export type UpdateCompanyInput = {
  CNPJ: string
  Name: string
  Environment: string
  UF: string
  SyncStartPolicy: SyncStartPolicy
  SyncStartDate: string
}

export type UpdateCredentialDataInput = {
  CredentialID: string
  Label: string
}

export type UpdateCredentialPathInput = {
  CredentialID: string
  CertPath: string
}

export type AssignCredentialInput = {
  CompanyCNPJ: string
  CredentialID: string
}

export type SyncSource = 'nfse' | 'nfe'

export type PullInput = {
  CNPJ: string
  Mode: string
}

export type PullResult = {
  CompanyName: string
  CNPJ: string
  CredentialLabel: string
  CredentialCNPJ: string
  ConsultationBasis: string
  Status: string
  StopReason: string
  LastProcessedNSU: number
  LastFoundNSU: number | null
  EmptyStreak: number
  DocumentsFound: number
  EventsFound: number
  DocumentsSaved: number
  EventsSaved: number
  DocumentsSkippedByPolicy: number
  EventsSkippedByPolicy: number
  Errors: number
  Duration: number
}

export type ResetSyncInput = {
  CompanyCNPJ: string
}

export type BuildInfo = {
  version: string
  commit: string
  date: string
}

export type ConnectionTestResult = {
  certLoaded: boolean
  certSubject: string
  certExpiration: string
  mtlsAccepted: boolean
  endpointReached: boolean
  responseCode: string
  responseDetail: string
  statusExplanation: string
}

// NF-e (modelo 55). Enum fields allow '' so an unknown backend value never
// reads as a real fiscal state.

export type NFeSituacao = 'autorizada' | 'denegada' | 'cancelada'
export type NFeCompleteness = 'resumo' | 'completa'
export type NFeManifestacao = 'nenhuma' | 'ciencia' | 'confirmada' | 'desconhecida' | 'nao_realizada'
export type NFeRole = 'destinatario' | 'emitente' | 'transportador' | 'autorizado' | 'none'
// NFeConclusiveTipo is the tpEvento code of a conclusive manifestação:
// 210200 confirmação, 210220 desconhecimento, 210240 operação não realizada.
export type NFeConclusiveTipo = '210200' | '210220' | '210240'
export type NFeEventOutcome = 'registrada' | 'ja_registrada' | 'rejeitada' | 'nao_enviada'
export type NFePendingKind = 'sem_ciencia' | 'sem_conclusiva'
export type NFeBlockedReason = 'caught_up' | 'consumo_indevido' | 'rate_budget'

export type ListNFeInput = {
  CNPJ: string
  Competence: string
  Situacao: NFeSituacao | ''
  Completeness: NFeCompleteness | ''
  Manifestacao: NFeManifestacao | ''
  Role: NFeRole | ''
  EmitenteCNPJ: string
  // ChavesAcesso limits the list to these notes; empty or absent lists all.
  ChavesAcesso?: string[]
}

export type NFeRow = {
  ID: string
  DocumentID: string
  ChaveAcesso: string
  Serie: string
  Numero: string
  IssueDate?: ISODateValue
  AuthorizedAt?: ISODateValue
  Protocolo: string
  TipoOperacao: string
  EmitenteCNPJ: string
  EmitenteName: string
  EmitenteIE: string
  DestinatarioCNPJ: string
  DestinatarioName: string
  TotalValue: number
  Situacao: NFeSituacao | ''
  Completeness: NFeCompleteness | ''
  Manifestacao: NFeManifestacao | ''
  ManifestacaoAt?: ISODateValue
  CienciaDue?: ISODateValue
  ConclusiveDue?: ISODateValue
  CompanyRole: NFeRole | ''
  EventCount: number
  FirstSyncedAt?: ISODateValue
  LastSyncedAt?: ISODateValue
  // DaysLeft counts calendar days until ConclusiveDue: 0 on the due day,
  // negative once it passed, null without a deadline.
  DaysLeft: number | null
  TacitlyConfirmed: boolean
  // The backend says why the note cannot receive each manifestação; '' when
  // it can.
  CienciaBlockReason: string
  ConclusiveBlockReason: string
}

export type NFeKeyInput = {
  CNPJ: string
  ChaveAcesso: string
}

export type NFeEvent = {
  ID: string
  TpEvento: string
  NSeqEvento: number
  Description: string
  EventAt?: ISODateValue
  RegisteredAt?: ISODateValue
  Protocolo: string
  CStat: string
  XMotivo: string
  Justificativa: string
  Correcao: string
  AutorCNPJ: string
  Completeness: NFeCompleteness | ''
  Registered: boolean
  SentByNanci: boolean
}

// NFePendingRow is a full NFeRow plus what the pending list adds.
export type NFePendingRow = NFeRow & {
  Kind: NFePendingKind | ''
  CienciaOverdue: boolean
}

export type NFePendingInput = {
  CNPJ: string
  DueWithinDays: number
}

export type RegisterNFeCienciaInput = {
  CNPJ: string
  ChavesAcesso: string[]
}

export type NFeSkipped = {
  ChaveAcesso: string
  Reason: string
}

export type NFeCienciaPlan = {
  Eligible: NFeRow[]
  Skipped: NFeSkipped[]
}

export type RegisterNFeManifestacaoInput = {
  CNPJ: string
  ChaveAcesso: string
  Tipo: NFeConclusiveTipo
  Justificativa: string
}

export type NFeEventResult = {
  ChaveAcesso: string
  TpEvento: string
  Status: NFeEventOutcome | ''
  CStat: string
  XMotivo: string
  Protocolo: string
  RegisteredAt?: ISODateValue
}

// NFeEventBatchResult holds one result per eligible chave, in the order sent.
export type NFeEventBatchResult = {
  Results: NFeEventResult[]
  Skipped: NFeSkipped[]
  Interrupted: string
}

// NFeResetResult is what ResetNFe removed. The manifestações sent stay as
// the audit trail.
export type NFeResetResult = {
  CompanyName: string
  CNPJ: string
  CompanyDocuments: number
  Documents: number
  Events: number
  ExportMarks: number
  ManifestacoesKept: number
}

export type PullNFeInput = {
  CNPJ: string
}

export type PullNFeResult = {
  CompanyName: string
  CNPJ: string
  Status: string
  StopReason: string
  UltNSU: number
  MaxNSU: number | null
  CompletasSaved: number
  ResumosSaved: number
  EventsSaved: number
  Errors: number
  NextAllowedAt?: ISODateValue
  RequestsLastHour: number
  RequestBudget: number
  Duration: number
}

export type NFeStatusResult = {
  CompanyName: string
  CNPJ: string
  UF: string
  TpAmb: string
  LastCheckedNSU: number
  MaxNSU: number | null
  LastSyncAt?: ISODateValue
  LastRunStatus: string
  LastRunStopReason: string
  InitialSyncDoneAt?: ISODateValue
  NextAllowedAt?: ISODateValue
  BlockedReason: NFeBlockedReason | ''
  RequestsLastHour: number
  RequestBudget: number
  TotalDestinatario: number
  TotalEmitente: number
  TotalOutros: number
  TotalResumos: number
  TotalCompletas: number
  PendingCiencia: number
  PendingConclusiva: number
  CienciaOverdue: number
}

export type ExportNFeXMLInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

export type ExportNFeZIPInput = {
  CNPJ: string
  Competence: string
  Role: NFeRole | ''
  ChavesAcesso: string[]
  IncludeResumos: boolean
  Incremental: boolean
  OutPath: string
}

export type NFeExportResult = ExportResult & {
  SkippedResumos: number
}
