import type { ExportResult, ISODateValue } from './common'

// NF-e (modelo 55). Enum fields allow '' so an unknown backend value never
// reads as a real fiscal state.

export type NFeSituacao = 'autorizada' | 'denegada' | 'cancelada'
export type NFeCompleteness = 'resumo' | 'completa'
export type NFeManifestacao =
  'nenhuma' | 'ciencia' | 'confirmada' | 'desconhecida' | 'nao_realizada'
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
  // OnlyUnread keeps the notes not yet marked as viewed.
  OnlyUnread?: boolean
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
  TpNF: string
  // "1" produção, "2" homologação.
  TpAmb: string
  // NatOp, ICMSValue and IPIValue are empty on a resumo.
  NatOp: string
  EmitenteCNPJ: string
  EmitenteName: string
  EmitenteIE: string
  EmitenteUF: string
  DestinatarioCNPJ: string
  DestinatarioName: string
  // Money fields are in cents.
  TotalValue: number
  ICMSValue: number
  IPIValue: number
  Situacao: NFeSituacao | ''
  Completeness: NFeCompleteness | ''
  LayoutVersion: string
  ParseWarnings: string[]
  Manifestacao: NFeManifestacao | ''
  ManifestacaoAt?: ISODateValue
  CienciaDue?: ISODateValue
  ConclusiveDue?: ISODateValue
  CompanyRole: NFeRole | ''
  EventCount: number
  // FirstSeenNSU is null for a note that came by XML import, not by the
  // distribution.
  FirstSeenNSU: number | null
  FirstSyncedAt?: ISODateValue
  LastSyncedAt?: ISODateValue
  // ViewedAt is empty while the note is new.
  ViewedAt?: ISODateValue
  // DaysLeft counts calendar days until ConclusiveDue: 0 on the due day,
  // negative once it passed, null without a deadline.
  DaysLeft: number | null
  // CienciaDaysLeft counts the same way until CienciaDue.
  CienciaDaysLeft: number | null
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
  LastNSU: number
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
  LastNSU: number
  MaxNSU: number | null
  LastSyncAt?: ISODateValue
  LastRunStatus: string
  LastRunStopReason: string
  LastRunError: string // set only when LastRunStatus is 'failed'
  InitialSyncDoneAt?: ISODateValue
  NextAllowedAt?: ISODateValue
  BlockedReason: NFeBlockedReason | ''
  RequestsLastHour: number
  RequestBudget: number
  // IdleDays is the number of days since the last successful distribution
  // query, set only when it nears the 60-day NSU rule; 0 otherwise.
  IdleDays: number
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

// NFeExportFormat is the NF-e list export the user picks in the dialog.
export type NFeExportFormat = 'csv' | 'xlsx' | 'zip' | 'danfe-zip'

export type ExportNFeDocumentsInput = {
  CNPJ: string
  Competence: string
  Role: NFeRole | ''
  Format: NFeExportFormat
  ChavesAcesso: string[]
  IncludeResumos: boolean
  Incremental: boolean
  OutPath: string
}

export type ExportDANFEInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

export type NFeExportResult = ExportResult & {
  SkippedResumos: number
}

export type FetchOutcome = 'completa' | 'resumo' | 'not_found'

// FetchByChaveResult is what SEFAZ delivered for one NF-e asked by its chave
// and the hourly budget after the request.
export type FetchByChaveResult = {
  ChaveAcesso: string
  Outcome: FetchOutcome | ''
  Message: string
  RequestsLastHour: number
  RequestBudget: number
}
