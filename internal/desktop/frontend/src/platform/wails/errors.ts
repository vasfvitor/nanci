// errors.ts reads the errors of the desktop backend. It imports neither the
// generated bindings nor the Wails runtime, so components and pages can use it.

// WailsErrorCode lists the codes set by formatError in internal/desktop/errors.go,
// which takes the core's from app.ErrorCode (internal/app/errorcode.go). Wails v3
// rejects a bound-method promise with a RuntimeError whose cause is the {code, message}
// payload marshalError builds.
const wailsErrorCodes = [
  'canceled',
  'sefaz_blocked',
  'sync_running',
  'no_requests_left',
  'not_found',
  'invalid_input',
  'wrong_password',
  'update_busy',
  'startup_failed',
] as const
export type WailsErrorCode = (typeof wailsErrorCodes)[number] | ''

type RawRecord = Record<string, unknown>

function asErrorCode(value: unknown): WailsErrorCode {
  return typeof value === 'string' && (wailsErrorCodes as readonly string[]).includes(value)
    ? (value as WailsErrorCode)
    : ''
}

function asText(value: unknown) {
  return typeof value === 'string' ? value : ''
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

// canceledMessage matches app.ErrOperationCanceled in Go.
const canceledMessage = 'operação cancelada pelo usuário'

export function normalizeError(error: unknown): WailsClientError {
  if (error instanceof WailsClientError) return error
  // The runtime rejects a cancelled call at once with a CancelError, while the
  // Go method runs until it sees its context end. Its cause is the abort
  // reason, which is not a payload of formatError.
  if (error instanceof Error && error.name === 'CancelError') {
    return new WailsClientError(canceledMessage, 'canceled', error)
  }
  if (error instanceof Error) {
    const cause = error.cause
    if (cause && typeof cause === 'object' && 'code' in cause) {
      const payload = cause as RawRecord
      const message = asText(payload['message']) || error.message
      return new WailsClientError(message, asErrorCode(payload['code']), error)
    }
    return new WailsClientError(error.message, '', error)
  }
  if (error && typeof error === 'object' && 'message' in error) {
    const payload = error as RawRecord
    return new WailsClientError(asText(payload['message']), asErrorCode(payload['code']), error)
  }
  return new WailsClientError(String(error), '', error)
}
