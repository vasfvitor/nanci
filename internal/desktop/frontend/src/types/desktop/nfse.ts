import type { ISODateValue } from './common'

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

// ExportFormat is the NFS-e list export the user picks in the dialog.
export type ExportFormat = 'csv' | 'xlsx' | 'zip' | 'danfse-zip'

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

export type QueryNFSeInput = {
  CompanyCNPJ: string
  ChaveAcesso: string
}

export type ExportXMLInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

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
