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
	docs := []nfe.CompanyDocument{
		{
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
		{
			Document: nfe.Document{
				ID:           "doc-2",
				Situacao:     nfe.SituacaoCancelada,
				Completeness: nfe.CompletenessResumo,
			},
			RelationID:   "rel-2",
			CompanyRole:  nfe.CompanyRoleNone,
			Manifestacao: nfe.ManifestacaoNaoRealizada,
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
	if row.CienciaDue == nil || !row.CienciaDue.Equal(authorized.AddDate(0, 0, nfe.CienciaWarningDays)) {
		t.Errorf("CienciaDue = %v, want authorization + %d days", row.CienciaDue, nfe.CienciaWarningDays)
	}
	if row.ConclusiveDue == nil || !row.ConclusiveDue.Equal(authorized.AddDate(0, 0, nfe.ConclusiveDeadlineDays)) {
		t.Errorf("ConclusiveDue = %v, want authorization + %d days", row.ConclusiveDue, nfe.ConclusiveDeadlineDays)
	}
	if row.EventCount != 2 {
		t.Errorf("EventCount = %d, want 2", row.EventCount)
	}

	empty := rows[1]
	if empty.AuthorizedAt != nil || empty.ManifestacaoAt != nil || empty.CienciaDue != nil || empty.ConclusiveDue != nil {
		t.Errorf("dates = %v %v %v %v, want all nil", empty.AuthorizedAt, empty.ManifestacaoAt, empty.CienciaDue, empty.ConclusiveDue)
	}
	if empty.Situacao != "cancelada" || empty.Completeness != "resumo" || empty.Manifestacao != "nao_realizada" || empty.CompanyRole != "none" {
		t.Errorf("enums = %q %q %q %q", empty.Situacao, empty.Completeness, empty.Manifestacao, empty.CompanyRole)
	}
}

func TestNFePendingRows(t *testing.T) {
	conclusive := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	rows := NFePendingRows([]app.NFePendingManifestation{{
		ChaveAcesso:    "35260912345678000195550010000123451000123456",
		Kind:           app.NFePendingSemCiencia,
		TotalValue:     nfse.NewMoneyFromCents(990),
		Completeness:   "resumo",
		Manifestacao:   "nenhuma",
		ConclusiveDue:  conclusive,
		DaysLeft:       -2,
		CienciaOverdue: true,
		Expired:        true,
	}})

	row := rows[0]
	if row.Kind != "sem_ciencia" || row.TotalValue != 990 || row.DaysLeft != -2 || !row.CienciaOverdue || !row.Expired {
		t.Errorf("row = %+v", row)
	}
	if row.Deadline == nil || !row.Deadline.Equal(conclusive) || row.ConclusiveDue == nil || !row.ConclusiveDue.Equal(conclusive) {
		t.Errorf("Deadline = %v, ConclusiveDue = %v, want %v", row.Deadline, row.ConclusiveDue, conclusive)
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
		Requested:         4,
		Registered:        1,
		AlreadyRegistered: 1,
		Rejected:          1,
		NotSent:           1,
		Outcomes: []app.NFeEventOutcome{
			{ChaveAcesso: "a", TpEvento: "210210", Status: app.NFeOutcomeRegistrada, CStat: "135", Protocolo: "p1", RegisteredAt: &registeredAt},
			{ChaveAcesso: "b", TpEvento: "210210", Status: app.NFeOutcomeJaRegistrada, CStat: "573"},
			{ChaveAcesso: "c", TpEvento: "210210", Status: app.NFeOutcomeRejeitada, CStat: "650", XMotivo: "Rejeição"},
			{ChaveAcesso: "d", TpEvento: "210210", Status: app.NFeOutcomeNaoEnviada},
		},
		Skipped:     []app.NFeSkipped{{ChaveAcesso: "e", Reason: "a empresa não é a destinatária"}},
		Interrupted: "timeout",
	})

	if got.Requested != 4 || got.Registered != 1 || got.AlreadyRegistered != 1 || got.Rejected != 1 || got.NotSent != 1 {
		t.Errorf("counts = %+v", got)
	}
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
		Eligible: []app.NFeCandidate{{ChaveAcesso: "a", TotalValue: nfse.NewMoneyFromCents(5050), CienciaDue: due}},
		Skipped:  []app.NFeSkipped{{ChaveAcesso: "b", Reason: "NF-e cancelada"}},
		Lotes:    1,
	})
	if len(got.Eligible) != 1 || got.Eligible[0].TotalValue != 5050 {
		t.Fatalf("Eligible = %+v", got.Eligible)
	}
	if got.Eligible[0].CienciaDue == nil || got.Eligible[0].ConclusiveDue != nil {
		t.Errorf("CienciaDue = %v, ConclusiveDue = %v", got.Eligible[0].CienciaDue, got.Eligible[0].ConclusiveDue)
	}
	if len(got.Skipped) != 1 || got.Skipped[0].Reason != "NF-e cancelada" || got.Lotes != 1 {
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
