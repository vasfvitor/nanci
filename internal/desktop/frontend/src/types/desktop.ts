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
export type NFeConclusiveTipo = 'confirmacao' | 'desconhecimento' | 'nao_realizada'
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
  EmitenteCNPJ?: string
}

export type NFeRow = {
  ID: string
  DocumentID: string
  ChaveAcesso: string
  Serie: string
  Numero: string
  IssueDate?: ISODateValue
  AuthorizedAt?: ISODateValue
  Protocol: string
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
}

export type NFeEvent = {
  ID: string
  TpEvento: string
  NSeqEvento: number
  EventAt?: ISODateValue
  RegisteredAt?: ISODateValue
  Protocolo: string
  CStat: string
  XMotivo: string
  Justificativa: string
  Correcao: string
  AutorCNPJ: string
  SentByNanci: boolean
}

export type NFePendingRow = NFeRow & {
  Kind: NFePendingKind | ''
  Deadline?: ISODateValue
  DaysLeft: number
  CienciaOverdue: boolean
  Expired: boolean
}

export type RegisterCienciaInput = {
  CNPJ: string
  ChavesAcesso: string[]
}

export type RegisterManifestationInput = {
  CNPJ: string
  ChaveAcesso: string
  Tipo: NFeConclusiveTipo
  Justificativa: string
}

export type NFeEventResult = {
  ChaveAcesso: string
  TipoEvento: string
  Status: NFeEventOutcome | ''
  CStat: string
  XMotivo: string
  Protocol: string
  RegisteredAt?: ISODateValue
}

export type NFeSkipped = {
  ChaveAcesso: string
  Reason: string
}

export type NFeEventBatchResult = {
  Results: NFeEventResult[]
  Requested: number
  Registered: number
  AlreadyRegistered: number
  Rejected: number
  Skipped: NFeSkipped[]
  Interrupted: string
}

export type PullNFeInput = {
  CNPJ: string
  Mode: string
}

export type PullNFeResult = PullResult & {
  Source: SyncSource | ''
  UltNSU: number
  MaxNSU: number
  NextAllowedAt: string | null
  RequestsLastHour: number
  RequestBudget: number
}

export type NFeStatusResult = {
  CompanyName: string
  CNPJ: string
  Environment: string
  LastCheckedNSU: number
  MaxNSU: number | null
  LastSyncAt?: ISODateValue
  LastRunStatus: string
  LastRunStopReason: string
  NextAllowedAt: string | null
  BlockedReason: NFeBlockedReason | ''
  RequestsLastHour: number
  RequestBudget: number
  InitialSyncDoneAt?: ISODateValue
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

export type ExportNFeZIPInput = ListNFeInput & {
  ChavesAcesso: string[]
  OutPath: string
}
