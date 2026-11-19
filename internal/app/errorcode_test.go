package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func TestErrorCode(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{errors.New("qualquer"), ""},
		{fmt.Errorf("carregar certificado: %w", ErrOperationCanceled), ErrorCodeCanceled},
		{context.Canceled, ErrorCodeCanceled},
		{&sync.BlockedError{Source: syncstate.SyncSourceNFe, Until: time.Now(), Reason: syncstate.SyncStopReasonCaughtUp}, ErrorCodeSefazBlocked},
		{fmt.Errorf("%w (NF-e) em outro processo do nanci (CLI)", ErrSyncRunning), ErrorCodeSyncRunning},
		{fmt.Errorf("%w (2 ficam)", sync.ErrNoRequestsLeft), ErrorCodeNoRequestsLeft},
		{company.ErrCompanyNotFound, ErrorCodeNotFound},
		{fmt.Errorf("buscar: %w", nfe.ErrDocumentNotFound), ErrorCodeNotFound},
		{fmt.Errorf("chave: %w", dfe.ErrInvalidAccessKey), ErrorCodeInvalidInput},
		{cnpj.ErrInvalidCheckDigits, ErrorCodeInvalidInput},
		{cert.ErrInvalidPass, ErrorCodeWrongPassword},
	}
	for _, tt := range tests {
		if got := ErrorCode(tt.err); got != tt.want {
			t.Errorf("ErrorCode(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}
