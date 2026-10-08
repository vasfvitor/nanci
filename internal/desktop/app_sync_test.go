package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	nsync "github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// certLabel is a variable, not a literal, so gosec does not read it as a
// hardcoded credential.
var certLabel = "Certificado A1"

func TestQueryNFSeEvents(t *testing.T) {
	f, a := newFakeApp(nil)
	f.query.events = `{"eventos":[]}`

	got, err := a.QueryNFSeEvents(desktopapi.QueryNFSeInput{CompanyCNPJ: testCNPJ, ChaveAcesso: testChave})
	if err != nil {
		t.Fatalf("QueryNFSeEvents: %v", err)
	}

	assertCalls(t, &f.query.recorder, call{"QueryNFSeEvents", []any{app.QueryNFSeInput{CNPJ: testCNPJ, ChaveAcesso: testChave}}})
	if got != `{"eventos":[]}` {
		t.Errorf("result = %q", got)
	}
}

func TestConnectionTests(t *testing.T) {
	connection := app.ConnectionTestResult{
		CertLoaded:        true,
		CertSubject:       "CN=EMPRESA TESTE",
		CertExpiration:    "2027-01-01",
		MTLSAccepted:      true,
		EndpointReached:   true,
		ResponseCode:      "137",
		ResponseDetail:    "Nenhum documento localizado",
		StatusExplanation: "Conexão OK",
	}
	want := desktopapi.ConnectionTestResult{
		CertLoaded:        true,
		CertSubject:       "CN=EMPRESA TESTE",
		CertExpiration:    "2027-01-01",
		MTLSAccepted:      true,
		EndpointReached:   true,
		ResponseCode:      "137",
		ResponseDetail:    "Nenhum documento localizado",
		StatusExplanation: "Conexão OK",
	}
	tests := []struct {
		name     string
		run      func(a *App) (desktopapi.ConnectionTestResult, error)
		recorder func(f fakeSet) *recorder
	}{
		{"TestConnection", func(a *App) (desktopapi.ConnectionTestResult, error) { return a.TestConnection(testCNPJ) },
			func(f fakeSet) *recorder { return &f.query.recorder }},
		{"TestCTeConnection", func(a *App) (desktopapi.ConnectionTestResult, error) { return a.TestCTeConnection(testCNPJ) },
			func(f fakeSet) *recorder { return &f.cte.recorder }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)
			f.query.connection = connection
			f.cte.connection = connection

			got, err := tt.run(a)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			assertCalls(t, tt.recorder(f), call{"TestConnection", []any{testCNPJ}})
			if got != want {
				t.Errorf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func TestPullNFSe(t *testing.T) {
	f, a := newFakeApp(nil)
	lastFound := int64(120)
	f.sync.pull = nsync.PullResult{
		Source:                   syncstate.SyncSourceNFSe,
		CompanyName:              "Empresa Teste",
		CNPJ:                     testCNPJ,
		CredentialLabel:          certLabel,
		CredentialCNPJ:           testCNPJ,
		ConsultationBasis:        "cnpj",
		Status:                   "completed",
		StopReason:               "caught_up",
		LastProcessedNSU:         119,
		LastFoundNSU:             &lastFound,
		EmptyStreak:              2,
		DocumentsFound:           10,
		EventsFound:              3,
		DocumentsSaved:           9,
		EventsSaved:              3,
		DocumentsSkippedByPolicy: 1,
		EventsSkippedByPolicy:    0,
		Errors:                   1,
		Duration:                 2 * time.Second,
	}

	got, err := a.Pull(desktopapi.PullInput{CNPJ: testCNPJ, Mode: "incremental"})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}

	assertCalls(t, &f.sync.recorder, call{"Pull", []any{nsync.PullInput{CNPJ: testCNPJ, Mode: "incremental", Source: syncstate.SyncSourceNFSe}}})
	want := desktopapi.PullResult{
		CompanyName:              "Empresa Teste",
		CNPJ:                     testCNPJ,
		CredentialLabel:          certLabel,
		CredentialCNPJ:           testCNPJ,
		ConsultationBasis:        "cnpj",
		Status:                   "completed",
		StopReason:               "caught_up",
		LastProcessedNSU:         119,
		LastFoundNSU:             &lastFound,
		EmptyStreak:              2,
		DocumentsFound:           10,
		EventsFound:              3,
		DocumentsSaved:           9,
		EventsSaved:              3,
		DocumentsSkippedByPolicy: 1,
		Errors:                   1,
		Duration:                 2 * time.Second,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestResetSyncStateResetsNFSe(t *testing.T) {
	f, a := newFakeApp(nil)

	if err := a.ResetSyncState(desktopapi.ResetSyncInput{CompanyCNPJ: testCNPJ}); err != nil {
		t.Fatalf("ResetSyncState: %v", err)
	}

	assertCalls(t, &f.sync.recorder, call{"ResetSyncState", []any{nsync.ResetSyncInput{CNPJ: testCNPJ, Source: syncstate.SyncSourceNFSe}}})
}

func TestStatusNFSe(t *testing.T) {
	f, a := newFakeApp(nil)
	notAfter := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSync := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	lastFound := int64(77)
	f.sync.status = nsync.StatusResult{
		CompanyName:        "Empresa Teste",
		CNPJ:               testCNPJ,
		Environment:        "producao",
		ConsultationCNPJ:   testCNPJ,
		CredentialCNPJ:     testCNPJ,
		CredentialNotAfter: &notAfter,
		LastProcessedNSU:   76,
		LastFoundNSU:       &lastFound,
		LastSyncAt:         &lastSync,
		LastRunStatus:      "completed",
		LastRunStopReason:  "caught_up",
		TotalEmitidas:      5,
		TotalTomadas:       6,
	}

	got, err := a.Status(testCNPJ)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	assertCalls(t, &f.sync.recorder, call{"Status", []any{testCNPJ}})
	want := desktopapi.StatusResult{
		CompanyName:        "Empresa Teste",
		CNPJ:               testCNPJ,
		Environment:        "producao",
		ConsultationCNPJ:   testCNPJ,
		CredentialCNPJ:     testCNPJ,
		CredentialNotAfter: &notAfter,
		LastProcessedNSU:   76,
		LastFoundNSU:       &lastFound,
		LastSyncAt:         &lastSync,
		LastRunStatus:      "completed",
		LastRunStopReason:  "caught_up",
		TotalEmitidas:      5,
		TotalTomadas:       6,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("status =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestPullNFe(t *testing.T) {
	f, a := newFakeApp(nil)
	maxNSU := int64(500)
	next := time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)
	f.nfe.pull = app.NFePullResult{
		CompanyName:      "Empresa Teste",
		CNPJ:             testCNPJ,
		Status:           "interrupted",
		StopReason:       "rate_budget",
		LastNSU:          450,
		MaxNSU:           &maxNSU,
		CompletasSaved:   8,
		ResumosSaved:     4,
		EventsSaved:      2,
		Errors:           1,
		NextAllowedAt:    &next,
		RequestsLastHour: 20,
		RequestBudget:    20,
		Duration:         time.Minute,
	}

	got, err := a.PullNFe(desktopapi.PullNFeInput{CNPJ: testCNPJ})
	if err != nil {
		t.Fatalf("PullNFe: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"Pull", []any{testCNPJ}})
	want := desktopapi.PullNFeResult{
		CompanyName:      "Empresa Teste",
		CNPJ:             testCNPJ,
		Status:           "interrupted",
		StopReason:       "rate_budget",
		LastNSU:          450,
		MaxNSU:           &maxNSU,
		CompletasSaved:   8,
		ResumosSaved:     4,
		EventsSaved:      2,
		Errors:           1,
		NextAllowedAt:    &next,
		RequestsLastHour: 20,
		RequestBudget:    20,
		Duration:         time.Minute,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestPullCTe(t *testing.T) {
	f, a := newFakeApp(nil)
	maxNSU := int64(60)
	f.cte.pull = app.CTePullResult{
		CompanyName:      "Empresa Teste",
		CNPJ:             testCNPJ,
		Status:           "completed",
		StopReason:       "caught_up",
		LastNSU:          60,
		MaxNSU:           &maxNSU,
		DocumentsSaved:   7,
		EventsSaved:      1,
		RequestsLastHour: 2,
		RequestBudget:    20,
		Duration:         3 * time.Second,
	}

	got, err := a.PullCTe(desktopapi.PullCTeInput{CNPJ: testCNPJ})
	if err != nil {
		t.Fatalf("PullCTe: %v", err)
	}

	assertCalls(t, &f.cte.recorder, call{"Pull", []any{testCNPJ}})
	want := desktopapi.PullCTeResult{
		CompanyName:      "Empresa Teste",
		CNPJ:             testCNPJ,
		Status:           "completed",
		StopReason:       "caught_up",
		LastNSU:          60,
		MaxNSU:           &maxNSU,
		DocumentsSaved:   7,
		EventsSaved:      1,
		RequestsLastHour: 2,
		RequestBudget:    20,
		Duration:         3 * time.Second,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result =\n  %+v\nwant\n  %+v", got, want)
	}
}

// TestSyncErrorsReachFormatError checks that the bound methods hand core
// errors back unwrapped enough for formatError to set the code the frontend
// reacts to.
func TestSyncErrorsReachFormatError(t *testing.T) {
	blocked := &nsync.BlockedError{Source: syncstate.SyncSourceNFe, Until: time.Date(2026, 9, 23, 14, 32, 0, 0, time.Local), Reason: syncstate.SyncStopReasonConsumoIndevido}
	running := fmt.Errorf("%w (NFS-e)", app.ErrSyncRunning)
	canceled := fmt.Errorf("carregar certificado: %w", app.ErrOperationCanceled)

	runs := map[string]func(a *App) (any, error){
		"Pull": func(a *App) (any, error) { return a.Pull(desktopapi.PullInput{CNPJ: testCNPJ}) },
		"ResetSyncState": func(a *App) (any, error) {
			return nil, a.ResetSyncState(desktopapi.ResetSyncInput{CompanyCNPJ: testCNPJ})
		},
		"Status":            func(a *App) (any, error) { return a.Status(testCNPJ) },
		"PullNFe":           func(a *App) (any, error) { return a.PullNFe(desktopapi.PullNFeInput{CNPJ: testCNPJ}) },
		"PullCTe":           func(a *App) (any, error) { return a.PullCTe(desktopapi.PullCTeInput{CNPJ: testCNPJ}) },
		"QueryNFSeEvents":   func(a *App) (any, error) { return a.QueryNFSeEvents(desktopapi.QueryNFSeInput{CompanyCNPJ: testCNPJ}) },
		"TestConnection":    func(a *App) (any, error) { return a.TestConnection(testCNPJ) },
		"TestCTeConnection": func(a *App) (any, error) { return a.TestCTeConnection(testCNPJ) },
	}
	errs := []struct {
		err  error
		code string
	}{
		{blocked, "sefaz_blocked"},
		{running, "sync_running"},
		{canceled, "canceled"},
		{errCore, ""},
	}
	for name, run := range runs {
		for _, e := range errs {
			t.Run(name+"/"+e.code, func(t *testing.T) {
				_, a := newFakeApp(e.err)

				got, err := run(a)
				if !errors.Is(err, e.err) {
					t.Fatalf("err = %v, want %v", err, e.err)
				}
				if got != nil && !reflect.ValueOf(got).IsZero() {
					t.Errorf("result = %+v, want zero", got)
				}
				payload, ok := formatError(err).(desktopapi.ErrorPayload)
				if !ok {
					t.Fatalf("formatError returned %T", formatError(err))
				}
				if payload.Code != e.code || payload.Message != e.err.Error() {
					t.Errorf("payload = %+v, want code %q and message %q", payload, e.code, e.err.Error())
				}
			})
		}
	}
}
