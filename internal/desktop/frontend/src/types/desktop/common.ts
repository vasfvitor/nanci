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
