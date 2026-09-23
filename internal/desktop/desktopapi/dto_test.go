package desktopapi

import (
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestNFeRows(t *testing.T) {
	authorized := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	cienciaDue := authorized.AddDate(0, 0, nfe.CienciaWarningDays)
	conclusiveDue := authorized.AddDate(0, 0, nfe.ConclusiveDeadlineDays)
	docs := []app.NFeDocument{
		{
			CompanyDocument: nfe.CompanyDocument{
				Document: nfe.Document{
					ID:           "doc-1",
					ChaveAcesso:  "35260912345678000195550010000123451000123456",
					Serie:        "1",
					Numero:       "12345",
					IssueDate:    authorized.Add(-time.Hour),
					AuthorizedAt: &authorized,
					Protocolo:    "135260000000001",
					TpNF:         "1",
					TotalValue:   nfse.NewMoneyFromCents(123456),
					Situacao:     nfe.SituacaoAutorizada,
					Completeness: nfe.CompletenessCompleta,
				},
				RelationID:   "rel-1",
				CompanyRole:  nfe.CompanyRoleDestinatario,
				Manifestacao: nfe.ManifestacaoCiencia,
				EventCount:   2,
			},
			Deadlines:          nfe.Deadlines{CienciaDue: cienciaDue, ConclusiveDue: conclusiveDue},
			DaysLeft:           -3,
			TacitlyConfirmed:   true,
			CienciaBlockReason: "já manifestada (ciencia)",
		},
		{
			CompanyDocument: nfe.CompanyDocument{
				Document: nfe.Document{
					ID:           "doc-2",
					Situacao:     nfe.SituacaoCancelada,
					Completeness: nfe.CompletenessResumo,
				},
				RelationID:   "rel-2",
				CompanyRole:  nfe.CompanyRoleNone,
				Manifestacao: nfe.ManifestacaoNaoRealizada,
			},
			CienciaBlockReason:    "a empresa não é a destinatária",
			ConclusiveBlockReason: "a empresa não é a destinatária",
		},
	}

	rows := NFeRows(docs)
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2", len(rows))
	}

	row := rows[0]
	if row.ID != "rel-1" || row.DocumentID != "doc-1" {
		t.Errorf("ID = %q, DocumentID = %q", row.ID, row.DocumentID)
	}
	if row.TotalValue != 123456 {
		t.Errorf("TotalValue = %d, want 123456 cents", row.TotalValue)
	}
	if row.TipoOperacao != "1" || row.Protocolo != "135260000000001" {
		t.Errorf("TipoOperacao = %q, Protocolo = %q", row.TipoOperacao, row.Protocolo)
	}
	if row.Situacao != "autorizada" || row.Completeness != "completa" || row.Manifestacao != "ciencia" || row.CompanyRole != "destinatario" {
		t.Errorf("enums = %q %q %q %q", row.Situacao, row.Completeness, row.Manifestacao, row.CompanyRole)
	}
	if row.CienciaDue == nil || !row.CienciaDue.Equal(cienciaDue) || row.ConclusiveDue == nil || !row.ConclusiveDue.Equal(conclusiveDue) {
		t.Errorf("CienciaDue = %v, ConclusiveDue = %v, want %v and %v", row.CienciaDue, row.ConclusiveDue, cienciaDue, conclusiveDue)
	}
	if row.DaysLeft == nil || *row.DaysLeft != -3 || !row.TacitlyConfirmed {
		t.Errorf("DaysLeft = %v, TacitlyConfirmed = %t; want -3 and true", row.DaysLeft, row.TacitlyConfirmed)
	}
	if row.CienciaBlockReason != "já manifestada (ciencia)" || row.ConclusiveBlockReason != "" {
		t.Errorf("block reasons = %q, %q", row.CienciaBlockReason, row.ConclusiveBlockReason)
	}
	if row.EventCount != 2 {
		t.Errorf("EventCount = %d, want 2", row.EventCount)
	}

	empty := rows[1]
	if empty.AuthorizedAt != nil || empty.ManifestacaoAt != nil || empty.CienciaDue != nil || empty.ConclusiveDue != nil || empty.DaysLeft != nil {
		t.Errorf("dates = %v %v %v %v, DaysLeft = %v; want all nil", empty.AuthorizedAt, empty.ManifestacaoAt, empty.CienciaDue, empty.ConclusiveDue, empty.DaysLeft)
	}
	if empty.Situacao != "cancelada" || empty.Completeness != "resumo" || empty.Manifestacao != "nao_realizada" || empty.CompanyRole != "none" {
		t.Errorf("enums = %q %q %q %q", empty.Situacao, empty.Completeness, empty.Manifestacao, empty.CompanyRole)
	}
	if empty.CienciaBlockReason == "" || empty.ConclusiveBlockReason == "" {
		t.Errorf("block reasons = %q, %q; want both set", empty.CienciaBlockReason, empty.ConclusiveBlockReason)
	}
}

func TestNFePendingRows(t *testing.T) {
	conclusive := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	rows := NFePendingRows([]app.NFePendingManifestation{{
		NFeDocument: app.NFeDocument{
			CompanyDocument: nfe.CompanyDocument{
				Document: nfe.Document{
					ChaveAcesso:  "35260912345678000195550010000123451000123456",
					TotalValue:   nfse.NewMoneyFromCents(990),
					Situacao:     nfe.SituacaoAutorizada,
					Completeness: nfe.CompletenessResumo,
				},
				RelationID:   "rel-1",
				CompanyRole:  nfe.CompanyRoleDestinatario,
				Manifestacao: nfe.ManifestacaoNenhuma,
			},
			Deadlines:        nfe.Deadlines{ConclusiveDue: conclusive},
			DaysLeft:         -2,
			TacitlyConfirmed: true,
		},
		Kind:           app.NFePendingSemCiencia,
		CienciaOverdue: true,
	}})

	row := rows[0]
	if row.Kind != "sem_ciencia" || row.ID != "rel-1" || row.TotalValue != 990 || !row.CienciaOverdue || !row.TacitlyConfirmed {
		t.Errorf("row = %+v", row)
	}
	if row.DaysLeft == nil || *row.DaysLeft != -2 {
		t.Errorf("DaysLeft = %v, want -2", row.DaysLeft)
	}
	if row.ConclusiveDue == nil || !row.ConclusiveDue.Equal(conclusive) {
		t.Errorf("ConclusiveDue = %v, want %v", row.ConclusiveDue, conclusive)
	}
	if row.CienciaDue != nil {
		t.Errorf("CienciaDue = %v, want nil for a zero date", row.CienciaDue)
	}
	if row.Situacao != "autorizada" || row.CompanyRole != "destinatario" {
		t.Errorf("Situacao = %q, CompanyRole = %q", row.Situacao, row.CompanyRole)
	}
}

func TestNFeEventBatch(t *testing.T) {
	registeredAt := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	got := NFeEventBatch(app.NFeManifestationSummary{
		Outcomes: []app.NFeEventOutcome{
			{ChaveAcesso: "a", TpEvento: "210210", Status: app.NFeOutcomeRegistrada, CStat: "135", Protocolo: "p1", RegisteredAt: &registeredAt},
			{ChaveAcesso: "b", TpEvento: "210210", Status: app.NFeOutcomeJaRegistrada, CStat: "573"},
			{ChaveAcesso: "c", TpEvento: "210210", Status: app.NFeOutcomeRejeitada, CStat: "650", XMotivo: "Rejeição"},
			{ChaveAcesso: "d", TpEvento: "210210", Status: app.NFeOutcomeNaoEnviada},
		},
		Skipped:     []app.NFeSkipped{{ChaveAcesso: "e", Reason: "a empresa não é a destinatária"}},
		Interrupted: "timeout",
	})

	if got.Interrupted != "timeout" {
		t.Errorf("Interrupted = %q", got.Interrupted)
	}
	wantStatus := []string{"registrada", "ja_registrada", "rejeitada", "nao_enviada"}
	if len(got.Results) != len(wantStatus) {
		t.Fatalf("len(Results) = %d, want %d", len(got.Results), len(wantStatus))
	}
	for i, want := range wantStatus {
		if got.Results[i].Status != want {
			t.Errorf("Results[%d].Status = %q, want %q", i, got.Results[i].Status, want)
		}
	}
	first := got.Results[0]
	if first.Protocolo != "p1" || first.CStat != "135" || first.RegisteredAt == nil || !first.RegisteredAt.Equal(registeredAt) {
		t.Errorf("Results[0] = %+v", first)
	}
	if len(got.Skipped) != 1 || got.Skipped[0].ChaveAcesso != "e" || got.Skipped[0].Reason == "" {
		t.Errorf("Skipped = %+v", got.Skipped)
	}
}

func TestNFeEventBatchEmptyListsAreNotNil(t *testing.T) {
	got := NFeEventBatch(app.NFeManifestationSummary{})
	if got.Results == nil || got.Skipped == nil {
		t.Errorf("Results = %v, Skipped = %v, want empty slices so the frontend gets []", got.Results, got.Skipped)
	}
}

func TestNFeCienciaPlanDTO(t *testing.T) {
	due := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got := NFeCienciaPlanDTO(app.NFeCienciaPlan{
		Eligible: []app.NFeDocument{{
			CompanyDocument: nfe.CompanyDocument{Document: nfe.Document{ChaveAcesso: "a", TotalValue: nfse.NewMoneyFromCents(5050)}},
			Deadlines:       nfe.Deadlines{CienciaDue: due},
		}},
		Skipped: []app.NFeSkipped{{ChaveAcesso: "b", Reason: "NF-e cancelada"}},
	})
	if len(got.Eligible) != 1 || got.Eligible[0].TotalValue != 5050 {
		t.Fatalf("Eligible = %+v", got.Eligible)
	}
	if got.Eligible[0].CienciaDue == nil || got.Eligible[0].ConclusiveDue != nil {
		t.Errorf("CienciaDue = %v, ConclusiveDue = %v", got.Eligible[0].CienciaDue, got.Eligible[0].ConclusiveDue)
	}
	if len(got.Skipped) != 1 || got.Skipped[0].Reason != "NF-e cancelada" {
		t.Errorf("plan = %+v", got)
	}
}

func TestNFeEvents(t *testing.T) {
	got := NFeEvents([]nfe.Event{{
		ID:            "ev-1",
		TpEvento:      "210240",
		NSeqEvento:    1,
		Protocolo:     "p",
		AutorCNPJ:     "12345678000195",
		Justificativa: "mercadoria não recebida",
		Completeness:  nfe.CompletenessCompleta,
		SentByNanci:   true,
	}})
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	ev := got[0]
	if ev.TpEvento != "210240" || ev.NSeqEvento != 1 || ev.Protocolo != "p" || ev.AutorCNPJ != "12345678000195" || ev.Justificativa == "" || ev.Completeness != "completa" || !ev.SentByNanci {
		t.Errorf("event = %+v", ev)
	}
	if ev.EventAt != nil || ev.RegisteredAt != nil {
		t.Errorf("EventAt = %v, RegisteredAt = %v, want nil", ev.EventAt, ev.RegisteredAt)
	}
}
