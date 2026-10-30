import type * as desktopapi from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'
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

// WailsErrorCode lists the codes set by formatError in internal/desktop/errors.go. Wails v3
// rejects a bound-method promise with a RuntimeError whose cause is the {code, message}
// payload marshalError builds.
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
  if (error instanceof Error) {
    const cause = error.cause
    if (cause && typeof cause === 'object' && 'code' in cause) {
      const payload = asRawRecord(cause)
      const message = asString(payload['message']) || error.message
      return new WailsClientError(message, asErrorCode(payload['code']), error)
    }
    return new WailsClientError(error.message, '', error)
  }
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

// asNullableNumber, asStringArray, asArray and asDate read the fields of the
// generated models that may be null: a Go pointer or a nil slice. They also
// check the value at runtime, so a malformed payload still yields a usable row.
export function asNullableNumber(value: number | null | undefined) {
  if (value === null || value === undefined) return null
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

export function asRawRecord(value: unknown): RawRecord {
  return value && typeof value === 'object' ? (value as RawRecord) : {}
}

export function asStringArray(value: string[] | null | undefined) {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []
}

export function asArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

export function asDate(value: string | null | undefined): ISODateValue {
  return typeof value === 'string' ? value : null
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

export function mapExportResult(item: desktopapi.ExportResult): ExportResult {
  return {
    OutPath: item.OutPath,
    Format: item.Format as ExportResult['Format'],
    Incremental: item.Incremental,
    ExportedCount: item.ExportedCount,
  }
}

function mapImportedFile(item: desktopapi.ImportedFile): ImportedFile {
  return {
    Path: item.Path,
    Kind: asEnum(item.Kind, importKinds),
    ChaveAcesso: item.ChaveAcesso,
    Outcome: asEnum(item.Outcome, importOutcomes),
    Message: item.Message,
  }
}

export function mapImportXMLResult(item: desktopapi.ImportXMLResult): ImportXMLResult {
  return {
    Files: asArray(item.Files).map(mapImportedFile),
    Imported: item.Imported,
    Updated: item.Updated,
    Rejected: item.Rejected,
  }
}

export function mapConnectionTestResult(
  item: desktopapi.ConnectionTestResult
): ConnectionTestResult {
  return {
    certLoaded: item.certLoaded,
    certSubject: item.certSubject,
    certExpiration: item.certExpiration,
    mtlsAccepted: item.mtlsAccepted,
    endpointReached: item.endpointReached,
    responseCode: item.responseCode,
    responseDetail: item.responseDetail,
    statusExplanation: item.statusExplanation,
  }
}
