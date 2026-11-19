package app

import (
	"context"
	"errors"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sync"
)

// Error codes a caller can branch on, the same in the desktop app and in the
// CLI's --json output. "" means no specific code: read the message.
const (
	ErrorCodeCanceled       = "canceled"         // the user or the caller cancelled
	ErrorCodeSefazBlocked   = "sefaz_blocked"    // the source must not be queried yet
	ErrorCodeSyncRunning    = "sync_running"     // the company's source is busy, here or in another process
	ErrorCodeNoRequestsLeft = "no_requests_left" // the hour's request budget has nothing left for it
	ErrorCodeNotFound       = "not_found"        // company, credential or document not found
	ErrorCodeInvalidInput   = "invalid_input"    // a malformed access key, CNPJ or option value
	ErrorCodeWrongPassword  = "wrong_password"   // the certificate password does not open it
)

// ErrorCode names what err is, by the sentinel it wraps, for callers that
// branch on it.
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrOperationCanceled), errors.Is(err, context.Canceled):
		return ErrorCodeCanceled
	case errors.Is(err, ErrSourceBlocked):
		return ErrorCodeSefazBlocked
	case errors.Is(err, ErrSyncRunning):
		return ErrorCodeSyncRunning
	case errors.Is(err, sync.ErrNoRequestsLeft):
		return ErrorCodeNoRequestsLeft
	case errors.Is(err, company.ErrCompanyNotFound), errors.Is(err, credential.ErrCredentialNotFound),
		errors.Is(err, nfe.ErrDocumentNotFound), errors.Is(err, cte.ErrDocumentNotFound):
		return ErrorCodeNotFound
	case errors.Is(err, dfe.ErrInvalidAccessKey), errors.Is(err, dfe.ErrInvalidEnum),
		errors.Is(err, cnpj.ErrInvalidLength), errors.Is(err, cnpj.ErrInvalidFormat),
		errors.Is(err, cnpj.ErrNonNumericCheckDigit), errors.Is(err, cnpj.ErrInvalidCheckDigits):
		return ErrorCodeInvalidInput
	case errors.Is(err, cert.ErrInvalidPass):
		return ErrorCodeWrongPassword
	}
	return ""
}
