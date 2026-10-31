package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/vasfvitor/nanci/internal/app"
	nsync "github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func TestFormatError(t *testing.T) {
	until := time.Date(2026, 9, 23, 14, 32, 0, 0, time.Local)
	blocked := &nsync.BlockedError{Source: syncstate.SyncSourceNFe, Until: until, Reason: syncstate.SyncStopReasonConsumoIndevido}

	tests := []struct {
		name     string
		err      error
		wantCode string
		wantText string
	}{
		{"canceled", fmt.Errorf("carregar certificado: %w", app.ErrOperationCanceled), "canceled", "operação cancelada"},
		{"call canceled", fmt.Errorf("sincronização: %w", context.Canceled), "canceled", "context canceled"},
		{"blocked", fmt.Errorf("pull: %w", blocked), "sefaz_blocked", "bloqueada até 23/09/2026 14:32"},
		{"blocked sentinel", app.ErrSourceBlocked, "sefaz_blocked", "bloqueada"},
		{"sync running", fmt.Errorf("%w (NF-e)", app.ErrSyncRunning), "sync_running", "já em andamento"},
		{"update during sync", errUpdateDuringSync, "update_busy", "aguarde a sincronização"},
		{"update busy", errUpdateBusy, "update_busy", "aguarde a operação em andamento"},
		{"updates off", errUpdatesDisabled, "", "indisponível"},
		{"update installing", errUpdateInstalling, "", "instalando uma atualização"},
		{"startup failed", fmt.Errorf("%w: %w", errNotStarted, errors.New("abrir banco de dados: disco cheio")), "startup_failed", "o nanci não iniciou: abrir banco de dados"},
		{"not started", errNotStarted, "startup_failed", "não iniciou"},
		{"plain", errors.New("empresa não encontrada"), "", "empresa não encontrada"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatError(tt.err)
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

func TestMarshalError(t *testing.T) {
	// Wails puts these bytes in the cause field of the rejected call;
	// client/base.ts reads these exact keys.
	data := marshalError(app.ErrOperationCanceled)
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("marshalError returned invalid JSON %q: %v", data, err)
	}
	if len(got) != 2 || got["code"] != "canceled" || got["message"] != app.ErrOperationCanceled.Error() {
		t.Errorf("json = %s, want code and message keys only", data)
	}
}

func TestBoundErrorCarriesCode(t *testing.T) {
	// Bindings.Add logs through the global Wails app, so one must exist; it
	// is never run.
	application.New(application.Options{Name: "nanci-test", Logger: slog.New(slog.DiscardHandler)})
	bindings := application.NewBindings(nil, nil)
	for _, service := range appServices(NewApp()) {
		if err := bindings.Add(service); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	// The test binary gives package main its import path, so the name is
	// built from the type instead of the "main.App" the app binds.
	name := reflect.TypeFor[App]().PkgPath() + ".App.ListCompanies"
	method := bindings.Get(&application.CallOptions{MethodName: name})
	if method == nil {
		t.Fatalf("%s is not bound", name)
	}

	// An App that never started answers errNotStarted.
	_, err := method.Call(context.Background(), nil)
	var callErr *application.CallError
	if !errors.As(err, &callErr) {
		t.Fatalf("Call error = %v (%T), want *application.CallError", err, err)
	}
	cause, ok := callErr.Cause.(json.RawMessage)
	if !ok {
		t.Fatalf("Cause = %T, want json.RawMessage", callErr.Cause)
	}
	var got map[string]string
	if err := json.Unmarshal(cause, &got); err != nil {
		t.Fatalf("Cause %q is not JSON: %v", cause, err)
	}
	if got["code"] != "startup_failed" {
		t.Errorf("cause = %s, want code startup_failed: the bound methods lost the App marshaler", cause)
	}
}
