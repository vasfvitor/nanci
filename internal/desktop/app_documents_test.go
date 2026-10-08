package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// documentFakes are the services behind the NFS-e, NF-e and CT-e pages.
type documentFakes struct {
	documents *fakeDocuments
	nfe       *fakeNFe
	cte       *fakeCTe
}

func newDocumentApp(err error) (documentFakes, *App) {
	f := documentFakes{
		documents: &fakeDocuments{err: err},
		nfe:       &fakeNFe{err: err},
		cte:       &fakeCTe{err: err},
	}
	return f, newTestApp(services{documents: f.documents, nfe: f.nfe, cte: f.cte})
}

func nfeDocument(chave string) app.NFeDocument {
	var d app.NFeDocument
	d.ChaveAcesso = dfe.AccessKey(chave)
	return d
}

func TestListNFeMapsFilters(t *testing.T) {
	f, a := newDocumentApp(nil)
	f.nfe.documents = []app.NFeDocument{nfeDocument(testChave)}

	got, err := a.ListNFe(desktopapi.ListNFeInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Situacao:     "autorizada",
		Completeness: "resumo",
		Manifestacao: "nenhuma",
		Role:         "destinatario",
		EmitenteCNPJ: "11222333000181",
		ChavesAcesso: []string{testChave},
		OnlyUnread:   true,
	})
	if err != nil {
		t.Fatalf("ListNFe: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"ListDocuments", []any{app.NFeListInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Situacao:     "autorizada",
		Completeness: "resumo",
		Role:         "destinatario",
		Manifestacao: "nenhuma",
		EmitenteCNPJ: "11222333000181",
		ChavesAcesso: []string{testChave},
		OnlyUnread:   true,
	}}})
	if len(got) != 1 || got[0].ChaveAcesso != testChave {
		t.Errorf("rows = %+v, want one row for %s", got, testChave)
	}
}

func TestListCTeMapsFilters(t *testing.T) {
	f, a := newDocumentApp(nil)
	var doc cte.CompanyDocument
	doc.ChaveAcesso = testChave
	f.cte.documents = []cte.CompanyDocument{doc}

	got, err := a.ListCTe(desktopapi.ListCTeInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Situacao:     "cancelada",
		Role:         "tomador",
		Modelo:       "57",
		EmitenteCNPJ: "11222333000181",
		TomadorCNPJ:  testCNPJ,
		NFeChave:     "nfe-chave",
		ChavesAcesso: []string{testChave},
		OnlyUnread:   true,
		Limit:        50,
	})
	if err != nil {
		t.Fatalf("ListCTe: %v", err)
	}

	assertCalls(t, &f.cte.recorder, call{"ListDocuments", []any{app.ListCTeInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Situacao:     "cancelada",
		Role:         "tomador",
		Modelo:       "57",
		EmitenteCNPJ: "11222333000181",
		TomadorCNPJ:  testCNPJ,
		NFeChave:     "nfe-chave",
		ChavesAcesso: []string{testChave},
		OnlyUnread:   true,
		Limit:        50,
	}}})
	if len(got) != 1 || got[0].ChaveAcesso != testChave {
		t.Errorf("rows = %+v, want one row for %s", got, testChave)
	}
}

func TestListDocumentsMapsFilters(t *testing.T) {
	f, a := newDocumentApp(nil)
	var doc nfse.CompanyDocument
	doc.ChaveAcesso = testChave
	f.documents.documents = []nfse.CompanyDocument{doc}

	got, err := a.ListDocuments(desktopapi.ListInput{CNPJ: testCNPJ, Competence: "2026-09", Direction: "tomada", OnlyUnread: true})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}

	assertCalls(t, &f.documents.recorder, call{"ListDocuments", []any{app.ListInput{
		CNPJ: testCNPJ, Competence: "2026-09", Direction: "tomada", OnlyUnread: true,
	}}})
	if len(got) != 1 || got[0].ChaveAcesso != testChave {
		t.Errorf("rows = %+v, want one row for %s", got, testChave)
	}
}

func TestListEvents(t *testing.T) {
	f, a := newDocumentApp(nil)
	f.documents.events = []app.EventView{{ID: "ev-nfse", Type: "cancelamento"}}
	f.nfe.events = []nfe.Event{{ID: "ev-nfe", TpEvento: "210210"}}
	f.cte.events = []cte.Event{{ID: "ev-cte", TpEvento: "110111"}}

	nfseEvents, err := a.ListEventsForDocument("doc-1")
	if err != nil || len(nfseEvents) != 1 || nfseEvents[0].ID != "ev-nfse" || nfseEvents[0].Type != "cancelamento" {
		t.Errorf("ListEventsForDocument = %+v, %v", nfseEvents, err)
	}
	nfeEvents, err := a.ListNFeEvents(desktopapi.NFeKeyInput{CNPJ: testCNPJ, ChaveAcesso: testChave})
	if err != nil || len(nfeEvents) != 1 || nfeEvents[0].ID != "ev-nfe" || nfeEvents[0].TpEvento != "210210" {
		t.Errorf("ListNFeEvents = %+v, %v", nfeEvents, err)
	}
	cteEvents, err := a.ListCTeEvents(desktopapi.CTeKeyInput{CNPJ: testCNPJ, ChaveAcesso: testChave})
	if err != nil || len(cteEvents) != 1 || cteEvents[0].ID != "ev-cte" || cteEvents[0].TpEvento != "110111" {
		t.Errorf("ListCTeEvents = %+v, %v", cteEvents, err)
	}

	assertCalls(t, &f.documents.recorder, call{"ListEventsForDocument", []any{"doc-1"}})
	assertCalls(t, &f.nfe.recorder, call{"ListEvents", []any{testCNPJ, testChave}})
	assertCalls(t, &f.cte.recorder, call{"ListEvents", []any{testCNPJ, testChave}})
}

func TestMarkViewed(t *testing.T) {
	chaves := []string{testChave, "outra-chave"}
	tests := []struct {
		name     string
		run      func(a *App) (int, error)
		recorder func(f documentFakes) *recorder
		method   string
	}{
		{
			name: "MarkDocumentsViewed",
			run: func(a *App) (int, error) {
				return a.MarkDocumentsViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ, ChavesAcesso: chaves})
			},
			recorder: func(f documentFakes) *recorder { return &f.documents.recorder },
			method:   "MarkDocumentsViewed",
		},
		{
			name: "MarkNFeViewed",
			run: func(a *App) (int, error) {
				return a.MarkNFeViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ, ChavesAcesso: chaves})
			},
			recorder: func(f documentFakes) *recorder { return &f.nfe.recorder },
			method:   "MarkViewed",
		},
		{
			name: "MarkCTeViewed",
			run: func(a *App) (int, error) {
				return a.MarkCTeViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ, ChavesAcesso: chaves})
			},
			recorder: func(f documentFakes) *recorder { return &f.cte.recorder },
			method:   "MarkViewed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newDocumentApp(nil)
			f.documents.viewed, f.nfe.viewed, f.cte.viewed = 2, 2, 2

			got, err := tt.run(a)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if got != 2 {
				t.Errorf("newly viewed = %d, want 2", got)
			}
			assertCalls(t, tt.recorder(f), call{tt.method, []any{testCNPJ, chaves}})
		})
	}
}

func TestStatusNFeMapsEveryField(t *testing.T) {
	f, a := newDocumentApp(nil)
	maxNSU := int64(900)
	lastSync := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	initial := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	next := time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)
	f.nfe.status = app.NFeStatusResult{
		SefazSourceStatus: app.SefazSourceStatus{
			CompanyName:       "Empresa Teste",
			CNPJ:              testCNPJ,
			UF:                "SP",
			TpAmb:             "1",
			LastNSU:           850,
			MaxNSU:            &maxNSU,
			LastSyncAt:        &lastSync,
			LastRunStatus:     "completed",
			LastRunStopReason: "caught_up",
			InitialSyncDoneAt: &initial,
			NextAllowedAt:     &next,
			BlockedReason:     "caught_up",
			RequestsLastHour:  3,
			RequestBudget:     20,
			IdleDays:          55,
		},
		TotalDestinatario: 10,
		TotalEmitente:     11,
		TotalOutros:       12,
		TotalResumos:      13,
		TotalCompletas:    14,
		PendingCiencia:    15,
		PendingConclusiva: 16,
		CienciaOverdue:    17,
	}

	got, err := a.StatusNFe(testCNPJ)
	if err != nil {
		t.Fatalf("StatusNFe: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"Status", []any{testCNPJ}})
	want := desktopapi.NFeStatusResult{
		CompanyName:       "Empresa Teste",
		CNPJ:              testCNPJ,
		UF:                "SP",
		TpAmb:             "1",
		LastNSU:           850,
		MaxNSU:            &maxNSU,
		LastSyncAt:        &lastSync,
		LastRunStatus:     "completed",
		LastRunStopReason: "caught_up",
		InitialSyncDoneAt: &initial,
		NextAllowedAt:     &next,
		BlockedReason:     "caught_up",
		RequestsLastHour:  3,
		RequestBudget:     20,
		IdleDays:          55,
		TotalDestinatario: 10,
		TotalEmitente:     11,
		TotalOutros:       12,
		TotalResumos:      13,
		TotalCompletas:    14,
		PendingCiencia:    15,
		PendingConclusiva: 16,
		CienciaOverdue:    17,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("status =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestStatusCTeMapsEveryField(t *testing.T) {
	f, a := newDocumentApp(nil)
	maxNSU := int64(40)
	lastSync := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	f.cte.status = app.CTeStatusResult{
		SefazSourceStatus: app.SefazSourceStatus{
			CompanyName:       "Empresa Teste",
			CNPJ:              testCNPJ,
			UF:                "PR",
			TpAmb:             "2",
			LastNSU:           38,
			MaxNSU:            &maxNSU,
			LastSyncAt:        &lastSync,
			LastRunStatus:     "failed",
			LastRunStopReason: "consumo_indevido",
			RequestsLastHour:  1,
			RequestBudget:     20,
			IdleDays:          61,
		},
		TotalTomador:      4,
		TotalDestinatario: 5,
		TotalRemetente:    6,
		TotalOutros:       7,
	}

	got, err := a.StatusCTe(testCNPJ)
	if err != nil {
		t.Fatalf("StatusCTe: %v", err)
	}

	assertCalls(t, &f.cte.recorder, call{"Status", []any{testCNPJ}})
	want := desktopapi.CTeStatusResult{
		CompanyName:       "Empresa Teste",
		CNPJ:              testCNPJ,
		UF:                "PR",
		TpAmb:             "2",
		LastNSU:           38,
		MaxNSU:            &maxNSU,
		LastSyncAt:        &lastSync,
		LastRunStatus:     "failed",
		LastRunStopReason: "consumo_indevido",
		RequestsLastHour:  1,
		RequestBudget:     20,
		IdleDays:          61,
		TotalTomador:      4,
		TotalDestinatario: 5,
		TotalRemetente:    6,
		TotalOutros:       7,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("status =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestListNFePendingManifestacoes(t *testing.T) {
	f, a := newDocumentApp(nil)
	f.nfe.pending = []app.NFePendingManifestacao{{NFeDocument: nfeDocument(testChave), Kind: "sem_ciencia", CienciaOverdue: true}}

	got, err := a.ListNFePendingManifestacoes(desktopapi.NFePendingInput{CNPJ: testCNPJ, DueWithinDays: 5})
	if err != nil {
		t.Fatalf("ListNFePendingManifestacoes: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"ListPendingManifestacoes", []any{app.NFePendingInput{CNPJ: testCNPJ, DueWithinDays: 5}}})
	if len(got) != 1 || got[0].ChaveAcesso != testChave || got[0].Kind != "sem_ciencia" || !got[0].CienciaOverdue {
		t.Errorf("rows = %+v", got)
	}
}

// TestPlanNFeCienciaSendsNothing: the plan feeds the confirmation dialog, so
// it must reach only PlanCiencia, never RegisterCiencia.
func TestPlanNFeCienciaSendsNothing(t *testing.T) {
	f, a := newDocumentApp(nil)
	f.nfe.plan = app.NFeCienciaPlan{
		Eligible: []app.NFeDocument{nfeDocument(testChave)},
		Skipped:  []app.NFeSkipped{{ChaveAcesso: "outra-chave", Reason: "já tem ciência"}},
	}

	got, err := a.PlanNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ, ChavesAcesso: []string{testChave, "outra-chave"}})
	if err != nil {
		t.Fatalf("PlanNFeCiencia: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"PlanCiencia", []any{app.NFeCienciaInput{CNPJ: testCNPJ, ChavesAcesso: []string{testChave, "outra-chave"}}}})
	if len(got.Eligible) != 1 || got.Eligible[0].ChaveAcesso != testChave {
		t.Errorf("eligible = %+v", got.Eligible)
	}
	if want := []desktopapi.NFeSkipped{{ChaveAcesso: "outra-chave", Reason: "já tem ciência"}}; !reflect.DeepEqual(got.Skipped, want) {
		t.Errorf("skipped = %+v, want %+v", got.Skipped, want)
	}
}

// TestRegisterNFeCienciaSendsOnlyTheSelection: the desktop never asks for
// every resumo, even when the selection is empty.
func TestRegisterNFeCienciaSendsOnlyTheSelection(t *testing.T) {
	registeredAt := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		chaves []string
	}{
		{"selection", []string{testChave}},
		{"empty selection", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newDocumentApp(nil)
			f.nfe.summary = app.NFeManifestacaoSummary{
				Outcomes:    []app.NFeEventOutcome{{ChaveAcesso: testChave, TpEvento: "210210", Status: "registrada", CStat: "135", XMotivo: "Evento registrado", Protocolo: "p-1", RegisteredAt: &registeredAt}},
				Skipped:     []app.NFeSkipped{{ChaveAcesso: "outra-chave", Reason: "cancelada"}},
				Interrupted: "falha de transporte",
			}

			got, err := a.RegisterNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ, ChavesAcesso: tt.chaves})
			if err != nil {
				t.Fatalf("RegisterNFeCiencia: %v", err)
			}

			assertCalls(t, &f.nfe.recorder, call{"RegisterCiencia", []any{app.NFeCienciaInput{CNPJ: testCNPJ, ChavesAcesso: tt.chaves}}})
			want := desktopapi.NFeEventBatchResult{
				Results:     []desktopapi.NFeEventResult{{ChaveAcesso: testChave, TpEvento: "210210", Status: "registrada", CStat: "135", XMotivo: "Evento registrado", Protocolo: "p-1", RegisteredAt: &registeredAt}},
				Skipped:     []desktopapi.NFeSkipped{{ChaveAcesso: "outra-chave", Reason: "cancelada"}},
				Interrupted: "falha de transporte",
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func TestRegisterNFeManifestacao(t *testing.T) {
	f, a := newDocumentApp(nil)
	f.nfe.outcome = app.NFeEventOutcome{ChaveAcesso: testChave, TpEvento: "210240", Status: "rejeitada", CStat: "573", XMotivo: "Duplicidade de evento"}

	got, err := a.RegisterNFeManifestacao(desktopapi.RegisterNFeManifestacaoInput{
		CNPJ:          testCNPJ,
		ChaveAcesso:   testChave,
		Tipo:          "nao_realizada",
		Justificativa: "Mercadoria não foi entregue",
	})
	if err != nil {
		t.Fatalf("RegisterNFeManifestacao: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"RegisterManifestacao", []any{app.NFeManifestacaoInput{
		CNPJ:          testCNPJ,
		ChaveAcesso:   testChave,
		Tipo:          "nao_realizada",
		Justificativa: "Mercadoria não foi entregue",
	}}})
	want := desktopapi.NFeEventResult{ChaveAcesso: testChave, TpEvento: "210240", Status: "rejeitada", CStat: "573", XMotivo: "Duplicidade de evento"}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
}

func TestResetNFe(t *testing.T) {
	f, a := newDocumentApp(nil)
	f.nfe.reset = app.NFeResetResult{
		CompanyName: "Empresa Teste",
		CNPJ:        testCNPJ,
		Environment: dfe.EnvironmentProduction,
		ResetCounts: nfe.ResetCounts{CompanyDocuments: 5, Documents: 4, Events: 3, ExportMarks: 2, ManifestacoesKept: 1},
	}

	got, err := a.ResetNFe(testCNPJ)
	if err != nil {
		t.Fatalf("ResetNFe: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"Reset", []any{testCNPJ}})
	want := desktopapi.NFeResetResult{CompanyName: "Empresa Teste", CNPJ: testCNPJ, CompanyDocuments: 5, Documents: 4, Events: 3, ExportMarks: 2, ManifestacoesKept: 1}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
}

func TestResetCTe(t *testing.T) {
	reset := app.CTeResetResult{
		CompanyName: "Empresa Teste",
		CNPJ:        testCNPJ,
		ResetCounts: cte.ResetCounts{CompanyDocuments: 5, Documents: 4, Events: 3, ExportMarks: 2},
	}
	want := desktopapi.CTeResetResult{CompanyName: "Empresa Teste", CNPJ: testCNPJ, CompanyDocuments: 5, Documents: 4, Events: 3, ExportMarks: 2}
	tests := []struct {
		method string
		run    func(a *App) (desktopapi.CTeResetResult, error)
	}{
		{"PreviewReset", func(a *App) (desktopapi.CTeResetResult, error) { return a.PreviewResetCTe(testCNPJ) }},
		{"Reset", func(a *App) (desktopapi.CTeResetResult, error) { return a.ResetCTe(testCNPJ) }},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			f, a := newDocumentApp(nil)
			f.cte.reset = reset

			got, err := tt.run(a)
			if err != nil {
				t.Fatalf("%s: %v", tt.method, err)
			}
			assertCalls(t, &f.cte.recorder, call{tt.method, []any{testCNPJ}})
			if got != want {
				t.Errorf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDocumentMethodsPassCoreErrors(t *testing.T) {
	runs := map[string]func(a *App) (any, error){
		"ListDocuments":         func(a *App) (any, error) { return a.ListDocuments(desktopapi.ListInput{CNPJ: testCNPJ}) },
		"ListEventsForDocument": func(a *App) (any, error) { return a.ListEventsForDocument("doc-1") },
		"MarkDocumentsViewed":   func(a *App) (any, error) { return a.MarkDocumentsViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ}) },
		"ListNFe":               func(a *App) (any, error) { return a.ListNFe(desktopapi.ListNFeInput{CNPJ: testCNPJ}) },
		"StatusNFe":             func(a *App) (any, error) { return a.StatusNFe(testCNPJ) },
		"MarkNFeViewed":         func(a *App) (any, error) { return a.MarkNFeViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ}) },
		"ListNFeEvents":         func(a *App) (any, error) { return a.ListNFeEvents(desktopapi.NFeKeyInput{CNPJ: testCNPJ}) },
		"ListNFePendingManifestacoes": func(a *App) (any, error) {
			return a.ListNFePendingManifestacoes(desktopapi.NFePendingInput{CNPJ: testCNPJ})
		},
		"PlanNFeCiencia": func(a *App) (any, error) {
			return a.PlanNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ})
		},
		"RegisterNFeCiencia": func(a *App) (any, error) {
			return a.RegisterNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ})
		},
		"RegisterNFeManifestacao": func(a *App) (any, error) {
			return a.RegisterNFeManifestacao(desktopapi.RegisterNFeManifestacaoInput{CNPJ: testCNPJ})
		},
		"ResetNFe":        func(a *App) (any, error) { return a.ResetNFe(testCNPJ) },
		"ListCTe":         func(a *App) (any, error) { return a.ListCTe(desktopapi.ListCTeInput{CNPJ: testCNPJ}) },
		"StatusCTe":       func(a *App) (any, error) { return a.StatusCTe(testCNPJ) },
		"MarkCTeViewed":   func(a *App) (any, error) { return a.MarkCTeViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ}) },
		"ListCTeEvents":   func(a *App) (any, error) { return a.ListCTeEvents(desktopapi.CTeKeyInput{CNPJ: testCNPJ}) },
		"PreviewResetCTe": func(a *App) (any, error) { return a.PreviewResetCTe(testCNPJ) },
		"ResetCTe":        func(a *App) (any, error) { return a.ResetCTe(testCNPJ) },
	}
	for name, run := range runs {
		t.Run(name, func(t *testing.T) {
			f, a := newDocumentApp(errCore)
			// Canned results that must not leak out next to an error.
			f.nfe.status.CNPJ = testCNPJ
			f.nfe.outcome.ChaveAcesso = testChave
			f.cte.status.CNPJ = testCNPJ
			f.cte.reset.CNPJ = testCNPJ
			f.nfe.reset.CNPJ = testCNPJ

			got, err := run(a)
			if !errors.Is(err, errCore) {
				t.Fatalf("err = %v, want %v", err, errCore)
			}
			if !reflect.ValueOf(got).IsZero() {
				t.Errorf("result = %+v, want zero", got)
			}
		})
	}
}
