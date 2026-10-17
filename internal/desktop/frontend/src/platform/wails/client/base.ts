import type {
  ConnectionTestResult,
  ExportResult,
  ImportedFile,
  ImportKind,
  ImportOutcome,
  ImportXMLResult,
  ISODateValue,
} from '@/types/desktop'

type RawRecord = Record<string, unknown>

// WailsErrorCode lists the codes set by formatError in internal/desktop/errors.go. The desktop
// ErrorFormatter rejects bound-method promises with {code, message}.
const wailsErrorCodes = ['canceled', 'sefaz_blocked', 'sync_running', 'startup_failed'] as const
export type WailsErrorCode = (typeof wailsErrorCodes)[number] | ''

function asErrorCode(value: unknown): WailsErrorCode {
  return asEnum(value, wailsErrorCodes)
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

export async function callWails<T>(operation: () => Promise<T>): Promise<T> {
  try {
    return await operation()
  } catch (error) {
    throw normalizeError(error)
  }
}

export function asString(value: unknown) {
  return typeof value === 'string' ? value : ''
}

export function asNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

export function asBoolean(value: unknown) {
  return typeof value === 'boolean' ? value : false
}

export function asNullableNumber(value: unknown) {
  if (value === null || value === undefined) return null
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

export function asRawRecord(value: unknown): RawRecord {
  return value && typeof value === 'object' ? (value as RawRecord) : {}
}

export function asStringArray(value: unknown) {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []
}

export function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

export function asDate(value: unknown): ISODateValue {
  if (typeof value === 'string' || value instanceof Date) return value
  return null
}

// asEnum keeps only known values. An unknown fiscal state becomes '' so the UI
// shows "Desconhecido" instead of guessing a real state.
export function asEnum<T extends string>(value: unknown, allowed: readonly T[]): T | '' {
  return typeof value === 'string' && (allowed as readonly string[]).includes(value)
    ? (value as T)
    : ''
}

const importKinds: readonly ImportKind[] = ['completa', 'resumo', 'evento']
const importOutcomes: readonly ImportOutcome[] = ['imported', 'updated', 'rejected']

// SaveDialog is what the save dialog of a list export shows: its title, the
// suggested file name and the file pattern.
export type SaveDialog = { title: string; defaultName: string; pattern: string }

export function fileTimestamp(now = new Date()) {
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  const h = String(now.getHours()).padStart(2, '0')
  const min = String(now.getMinutes()).padStart(2, '0')
  const s = String(now.getSeconds()).padStart(2, '0')
  return `${y}_${m}_${d}_${h}${min}${s}`
}

export function mapExportResult(raw: unknown): ExportResult {
  const item = asRawRecord(raw)
  return {
    OutPath: asString(item['OutPath']),
    Format: asString(item['Format']) as ExportResult['Format'],
    Incremental: asBoolean(item['Incremental']),
    ExportedCount: asNumber(item['ExportedCount']),
  }
}

function mapImportedFile(raw: unknown): ImportedFile {
  const item = asRawRecord(raw)
  return {
    Path: asString(item['Path']),
    Kind: asEnum(item['Kind'], importKinds),
    ChaveAcesso: asString(item['ChaveAcesso']),
    Outcome: asEnum(item['Outcome'], importOutcomes),
    Message: asString(item['Message']),
  }
}

export function mapImportXMLResult(raw: unknown): ImportXMLResult {
  const item = asRawRecord(raw)
  return {
    Files: asArray(item['Files']).map(mapImportedFile),
    Imported: asNumber(item['Imported']),
    Updated: asNumber(item['Updated']),
    Rejected: asNumber(item['Rejected']),
  }
}

export function mapConnectionTestResult(raw: unknown): ConnectionTestResult {
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
