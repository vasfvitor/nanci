package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

func TestDesktopError(t *testing.T) {
	until := time.Date(2026, 9, 23, 14, 32, 0, 0, time.Local)
	blocked := &nsync.BlockedError{Source: nfse.SyncSourceNFe, Until: until, Reason: nfse.SyncStopReasonConsumoIndevido}

	tests := []struct {
		name       string
		err        error
		wantPrefix string
		wantText   string
		wantIs     error
	}{
		{"canceled", fmt.Errorf("carregar certificado: %w", app.ErrOperationCanceled), "ERR_CANCELED: ", "operação cancelada", app.ErrOperationCanceled},
		{"blocked", fmt.Errorf("pull: %w", blocked), "ERR_SEFAZ_BLOCKED: ", "bloqueada até 23/09/2026 14:32", app.ErrSourceBlocked},
		{"blocked sentinel", app.ErrSourceBlocked, "ERR_SEFAZ_BLOCKED: ", "bloqueada", app.ErrSourceBlocked},
		{"sync running", fmt.Errorf("%w (NF-e)", app.ErrSyncRunning), "ERR_SYNC_RUNNING: ", "já em andamento", app.ErrSyncRunning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := desktopError(tt.err)
			if got == nil {
				t.Fatal("got nil")
			}
			msg := got.Error()
			if !strings.HasPrefix(msg, tt.wantPrefix) {
				t.Errorf("message %q, want prefix %q", msg, tt.wantPrefix)
			}
			if !strings.Contains(msg, tt.wantText) {
				t.Errorf("message %q, want it to contain %q", msg, tt.wantText)
			}
			if !errors.Is(got, tt.wantIs) {
				t.Errorf("errors.Is(%v, %v) = false", got, tt.wantIs)
			}
		})
	}
}

func TestDesktopErrorBlockedKeepsDetails(t *testing.T) {
	blocked := &nsync.BlockedError{Source: nfse.SyncSourceNFe, Until: time.Now().Add(time.Hour)}
	var target *nsync.BlockedError
	if !errors.As(desktopError(blocked), &target) || !target.Until.Equal(blocked.Until) {
		t.Errorf("errors.As lost the *BlockedError")
	}
}

func TestDesktopErrorNil(t *testing.T) {
	if err := desktopError(nil); err != nil {
		t.Errorf("desktopError(nil) = %v, want nil", err)
	}
}

func TestDesktopErrorPassthrough(t *testing.T) {
	plain := errors.New("empresa não encontrada")
	got := desktopError(plain)
	if !errors.Is(got, plain) || got.Error() != plain.Error() {
		t.Errorf("desktopError(plain) = %v, want it unchanged", got)
	}
}

func TestDesktopErrorCanceledMatchesPullOutput(t *testing.T) {
	// CompaniesPage matches "ERR_CANCELED" in the message; Pull used to build
	// it inline with this exact format.
	want := fmt.Errorf("ERR_CANCELED: %w", app.ErrOperationCanceled).Error()
	if got := desktopError(app.ErrOperationCanceled).Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
