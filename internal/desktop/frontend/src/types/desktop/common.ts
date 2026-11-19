export type ISODateValue = string | Date | null | undefined

// ExportResultFormat is what the backend reports it wrote.
export type ExportResultFormat = 'csv' | 'xlsx' | 'xml' | 'danfse' | 'danfe' | 'dacte'

export type ExportResult = {
  OutPath: string
  Format: ExportResultFormat
  Incremental: boolean
  ExportedCount: number
}

export type SyncSource = 'nfse' | 'nfe' | 'cte'

// RecoverNSUStopReason says why an NSU recovery stopped with NSUs left to ask
// for: its share of the hour, the hour's budget, a SEFAZ block (656), or the
// user's cancel.
export type RecoverNSUStopReason = 'limit' | 'rate_budget' | 'consumo_indevido' | 'canceled'

// RecoverNSUResult counts what one NF-e or CT-e NSU recovery (consNSU) did.
export type RecoverNSUResult = {
  Asked: number
  Recovered: number // NSUs whose document was stored
  Empty: number // NSUs with nothing to store (never generated, rejected alone or skipped)
  Unreadable: number // NSUs whose document could not be read; see the log
  Expired: number // NSUs closed unasked, past the 90-day window
  Missing: number // NSUs still missing after the run
  StopReason: RecoverNSUStopReason | ''
  NextAllowedAt?: ISODateValue
  RequestsLastHour: number
  RequestBudget: number
}

// BuildInfo.webview2Version is empty off Windows or when the runtime version
// cannot be read.
export type BuildInfo = {
  version: string
  commit: string
  date: string
  webview2Version: string
}

// UpdateInfo answers checkForUpdate. enabled is false in a build that does
// not update itself (a dev build, or one without the public key); version and
// notes describe the release found when available is true.
export type UpdateInfo = {
  enabled: boolean
  currentVersion: string
  available: boolean
  version: string
  notes: string
}

// UpdateProgress is the download progress the Wails updater emits, in bytes;
// total is 0 when the size is unknown.
export type UpdateProgress = {
  written: number
  total: number
  rate: number
  provider?: string
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

// --- Import ---

export type ImportKind = 'completa' | 'resumo' | 'evento'
export type ImportOutcome = 'imported' | 'updated' | 'rejected'

// ImportedFile is the result of one imported XML. Kind and ChaveAcesso are
// empty when the file was rejected before they were known.
export type ImportedFile = {
  Path: string
  Kind: ImportKind | ''
  ChaveAcesso: string
  Outcome: ImportOutcome | ''
  Message: string
}

// ImportXMLResult has one ImportedFile per picked file, in the order picked,
// and the counts per outcome.
export type ImportXMLResult = {
  Files: ImportedFile[]
  Imported: number
  Updated: number
  Rejected: number
}
