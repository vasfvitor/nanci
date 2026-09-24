package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/nfse"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

func TestFormatError(t *testing.T) {
	until := time.Date(2026, 9, 23, 14, 32, 0, 0, time.Local)
	blocked := &nsync.BlockedError{Source: nfse.SyncSourceNFe, Until: until, Reason: nfse.SyncStopReasonConsumoIndevido}

	tests := []struct {
		name     string
		err      error
		wantCode string
		wantText string
	}{
		{"canceled", fmt.Errorf("carregar certificado: %w", app.ErrOperationCanceled), "canceled", "operação cancelada"},
		{"blocked", fmt.Errorf("pull: %w", blocked), "sefaz_blocked", "bloqueada até 23/09/2026 14:32"},
		{"blocked sentinel", app.ErrSourceBlocked, "sefaz_blocked", "bloqueada"},
		{"sync running", fmt.Errorf("%w (NF-e)", app.ErrSyncRunning), "sync_running", "já em andamento"},
		{"plain", errors.New("empresa não encontrada"), "", "empresa não encontrada"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := formatError(tt.err).(desktopapi.ErrorPayload)
			if !ok {
				t.Fatalf("formatError returned %T, want desktopapi.ErrorPayload", formatError(tt.err))
			}
			if got.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.Message != tt.err.Error() {
				t.Errorf("Message = %q, want the original message %q", got.Message, tt.err.Error())
			}
			if !strings.Contains(got.Message, tt.wantText) {
				t.Errorf("Message = %q, want it to contain %q", got.Message, tt.wantText)
			}
		})
	}
}

func TestFormatErrorJSON(t *testing.T) {
	// The dispatcher marshals the payload into the callback's error field;
	// client.ts reads these exact keys.
	data, err := json.Marshal(formatError(app.ErrOperationCanceled))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["code"] != "canceled" || got["message"] != app.ErrOperationCanceled.Error() {
		t.Errorf("json = %s, want code and message keys only", data)
	}
}
