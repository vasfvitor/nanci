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

const testXML = "C:/xml/nota.xml"

func TestImportXML(t *testing.T) {
	core := app.ImportXMLResult{
		Files: []app.ImportedFile{
			{Path: testXML, Kind: "completa", ChaveAcesso: testChave, Outcome: "imported"},
			{Path: "C:/xml/evento.xml", Kind: "evento", ChaveAcesso: testChave, Outcome: "updated"},
			{Path: "C:/xml/outro.xml", Outcome: "rejected", Message: "o XML não cita a empresa"},
		},
		Imported: 1,
		Updated:  1,
		Rejected: 1,
	}
	want := desktopapi.ImportXMLResult{
		Files: []desktopapi.ImportedFile{
			{Path: testXML, Kind: "completa", ChaveAcesso: testChave, Outcome: "imported"},
			{Path: "C:/xml/evento.xml", Kind: "evento", ChaveAcesso: testChave, Outcome: "updated"},
			{Path: "C:/xml/outro.xml", Outcome: "rejected", Message: "o XML não cita a empresa"},
		},
		Imported: 1,
		Updated:  1,
		Rejected: 1,
	}
	files := []string{testXML, "C:/xml/evento.xml", "C:/xml/outro.xml"}
	wantCall := call{"ImportXML", []any{app.ImportXMLInput{CNPJ: testCNPJ, Files: files}}}
	input := desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: files}

	t.Run("nfe", func(t *testing.T) {
		f, a := newFakeApp(nil)
		f.nfe.imported = core

		got, err := a.ImportNFeXML(input)
		if err != nil {
			t.Fatalf("ImportNFeXML: %v", err)
		}
		assertCalls(t, &f.nfe.recorder, wantCall)
		assertCalls(t, &f.cte.recorder)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("result =\n  %+v\nwant\n  %+v", got, want)
		}
	})
	t.Run("cte", func(t *testing.T) {
		f, a := newFakeApp(nil)
		f.cte.imported = core

		got, err := a.ImportCTeXML(input)
		if err != nil {
			t.Fatalf("ImportCTeXML: %v", err)
		}
		assertCalls(t, &f.cte.recorder, wantCall)
		assertCalls(t, &f.nfe.recorder)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("result =\n  %+v\nwant\n  %+v", got, want)
		}
	})
}

func TestFetchNFeByChave(t *testing.T) {
	f, a := newFakeApp(nil)
	f.nfe.fetched = app.FetchByChaveResult{
		ChaveAcesso:      testChave,
		Outcome:          "resumo",
		Message:          "Só o resumo veio; registre a ciência para receber a NF-e completa.",
		RequestsLastHour: 3,
		RequestBudget:    20,
	}

	got, err := a.FetchNFeByChave(desktopapi.FetchByChaveInput{CNPJ: testCNPJ, ChaveAcesso: testChave})
	if err != nil {
		t.Fatalf("FetchNFeByChave: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"FetchByChave", []any{app.FetchByChaveInput{CNPJ: testCNPJ, ChaveAcesso: testChave}}})
	want := desktopapi.FetchByChaveResult{
		ChaveAcesso:      testChave,
		Outcome:          "resumo",
		Message:          "Só o resumo veio; registre a ciência para receber a NF-e completa.",
		RequestsLastHour: 3,
		RequestBudget:    20,
	}
	if got != want {
		t.Errorf("result =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestImportAndFetchRejectBeforeCore(t *testing.T) {
	tests := []struct {
		name    string
		run     func(a *App) (any, error)
		wantErr string
	}{
		{"nfe import without company", func(a *App) (any, error) {
			return a.ImportNFeXML(desktopapi.ImportXMLInput{Files: []string{testXML}})
		}, "empresa não especificada"},
		{"nfe import without files", func(a *App) (any, error) {
			return a.ImportNFeXML(desktopapi.ImportXMLInput{CNPJ: testCNPJ})
		}, "nenhum arquivo XML selecionado"},
		{"cte import without company", func(a *App) (any, error) {
			return a.ImportCTeXML(desktopapi.ImportXMLInput{Files: []string{testXML}})
		}, "empresa não especificada"},
		{"cte import without files", func(a *App) (any, error) {
			return a.ImportCTeXML(desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: []string{}})
		}, "nenhum arquivo XML selecionado"},
		{"fetch without company", func(a *App) (any, error) {
			return a.FetchNFeByChave(desktopapi.FetchByChaveInput{ChaveAcesso: testChave})
		}, "empresa não especificada"},
		{"fetch without chave", func(a *App) (any, error) {
			return a.FetchNFeByChave(desktopapi.FetchByChaveInput{CNPJ: testCNPJ})
		}, "chave de acesso não especificada"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			got, err := tt.run(a)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if !reflect.ValueOf(got).IsZero() {
				t.Errorf("result = %+v, want zero", got)
			}
			f.assertNoCalls(t)
		})
	}
}

// TestImportAndFetchPassCoreErrors checks that core errors come back with
// the code formatError gives them, and that the close guard is released.
func TestImportAndFetchPassCoreErrors(t *testing.T) {
	blocked := &nsync.BlockedError{Source: syncstate.SyncSourceNFe, Until: time.Date(2026, 9, 23, 14, 32, 0, 0, time.Local), Reason: syncstate.SyncStopReasonRateBudget}
	running := fmt.Errorf("%w (NF-e)", app.ErrSyncRunning)
	canceled := fmt.Errorf("carregar certificado: %w", app.ErrOperationCanceled)

	runs := map[string]func(a *App) (any, error){
		"ImportNFeXML": func(a *App) (any, error) {
			return a.ImportNFeXML(desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: []string{testXML}})
		},
		"ImportCTeXML": func(a *App) (any, error) {
			return a.ImportCTeXML(desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: []string{testXML}})
		},
		"FetchNFeByChave": func(a *App) (any, error) {
			return a.FetchNFeByChave(desktopapi.FetchByChaveInput{CNPJ: testCNPJ, ChaveAcesso: testChave})
		},
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
				if !reflect.ValueOf(got).IsZero() {
					t.Errorf("result = %+v, want zero", got)
				}
				payload, ok := formatError(err).(desktopapi.ErrorPayload)
				if !ok {
					t.Fatalf("formatError returned %T", formatError(err))
				}
				if payload.Code != e.code {
					t.Errorf("code = %q, want %q", payload.Code, e.code)
				}
				if n := a.guard.pulls.Load(); n != 0 {
					t.Errorf("pulls = %d after the error, want 0", n)
				}
			})
		}
	}
}
