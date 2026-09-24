package desktopapi

import (
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfe"
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
					TotalValue:   dfe.NewMoneyFromCents(123456),
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
			CienciaDaysLeft:    -80,
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
	if row.TpNF != "1" || row.Protocolo != "135260000000001" {
		t.Errorf("TpNF = %q, Protocolo = %q", row.TpNF, row.Protocolo)
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
	if row.CienciaDaysLeft == nil || *row.CienciaDaysLeft != -80 {
		t.Errorf("CienciaDaysLeft = %v, want -80", row.CienciaDaysLeft)
	}
	if row.CienciaBlockReason != "já manifestada (ciencia)" || row.ConclusiveBlockReason != "" {
		t.Errorf("block reasons = %q, %q", row.CienciaBlockReason, row.ConclusiveBlockReason)
	}
	if row.EventCount != 2 {
		t.Errorf("EventCount = %d, want 2", row.EventCount)
	}

	empty := rows[1]
	if empty.AuthorizedAt != nil || empty.ManifestacaoAt != nil || empty.CienciaDue != nil || empty.ConclusiveDue != nil || empty.DaysLeft != nil || empty.CienciaDaysLeft != nil {
		t.Errorf("dates = %v %v %v %v, days left = %v %v; want all nil", empty.AuthorizedAt, empty.ManifestacaoAt, empty.CienciaDue, empty.ConclusiveDue, empty.DaysLeft, empty.CienciaDaysLeft)
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
	rows := NFePendingRows([]app.NFePendingManifestacao{{
		NFeDocument: app.NFeDocument{
			CompanyDocument: nfe.CompanyDocument{
				Document: nfe.Document{
					ChaveAcesso:  "35260912345678000195550010000123451000123456",
					TotalValue:   dfe.NewMoneyFromCents(990),
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

func TestNFeEventResults(t *testing.T) {
	registeredAt := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	got := NFeEventResults(app.NFeManifestacaoSummary{
		Outcomes: []app.NFeEventOutcome{
			{ChaveAcesso: "a", TpEvento: "210210", Status: nfe.ManifestacaoStatusRegistrada, CStat: "135", Protocolo: "p1", RegisteredAt: &registeredAt},
			{ChaveAcesso: "b", TpEvento: "210210", Status: nfe.ManifestacaoStatusJaRegistrada, CStat: "573"},
			{ChaveAcesso: "c", TpEvento: "210210", Status: nfe.ManifestacaoStatusRejeitada, CStat: "650", XMotivo: "Rejeição"},
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

func TestNFeEventResultsEmptyListsAreNotNil(t *testing.T) {
	got := NFeEventResults(app.NFeManifestacaoSummary{})
	if got.Results == nil || got.Skipped == nil {
		t.Errorf("Results = %v, Skipped = %v, want empty slices so the frontend gets []", got.Results, got.Skipped)
	}
}

func TestNFeEventResultFrom(t *testing.T) {
	registeredAt := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	outcome := app.NFeEventOutcome{
		ChaveAcesso:  "a",
		TpEvento:     "210200",
		Status:       "registrada",
		CStat:        "135",
		XMotivo:      "Evento registrado",
		Protocolo:    "p1",
		RegisteredAt: &registeredAt,
	}
	got := NFeEventResultFrom(outcome)
	want := NFeEventResult{
		ChaveAcesso:  "a",
		TpEvento:     "210200",
		Status:       "registrada",
		CStat:        "135",
		XMotivo:      "Evento registrado",
		Protocolo:    "p1",
		RegisteredAt: &registeredAt,
	}
	if got != want {
		t.Errorf("NFeEventResultFrom = %+v, want %+v", got, want)
	}
}

func TestNFeCienciaPlanFrom(t *testing.T) {
	due := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got := NFeCienciaPlanFrom(app.NFeCienciaPlan{
		Eligible: []app.NFeDocument{{
			CompanyDocument: nfe.CompanyDocument{Document: nfe.Document{ChaveAcesso: "a", TotalValue: dfe.NewMoneyFromCents(5050)}},
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

func TestCTeRows(t *testing.T) {
	authorized := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	nsu := int64(42)
	docs := []cte.CompanyDocument{
		{
			Document: cte.Document{
				ID:                  "doc-1",
				ChaveAcesso:         "35260912345678000195570010000123451000123456",
				TpAmb:               "2",
				Modelo:              "57",
				TipoDocumento:       cte.TipoDocumentoCTe,
				Serie:               "1",
				Numero:              "12345",
				CFOP:                "5353",
				NatOp:               "Prestação de serviço de transporte",
				IssueDate:           authorized.Add(-time.Hour),
				Competence:          "2026-09",
				AuthorizedAt:        &authorized,
				Protocolo:           "135260000000001",
				TpCTe:               "0",
				TpServ:              "2",
				Modal:               "01",
				MunIni:              cte.Municipio{Codigo: "3550308", Nome: "São Paulo", UF: "SP"},
				MunFim:              cte.Municipio{Codigo: "3304557", Nome: "Rio de Janeiro", UF: "RJ"},
				Emitente:            cte.Party{CNPJ: "11111111000111", Name: "Transportadora"},
				Remetente:           cte.Party{CNPJ: "22222222000122", Name: "Remetente"},
				Destinatario:        cte.Party{CNPJ: "33333333000133", Name: "Destinatário"},
				Expedidor:           cte.Party{CNPJ: "44444444000144", Name: "Expedidor"},
				Recebedor:           cte.Party{CNPJ: "55555555000155", Name: "Recebedor"},
				Tomador:             cte.Party{CNPJ: "22222222000122", Name: "Remetente", IE: "123", UF: "SP"},
				TomadorIndicador:    "0",
				TotalValue:          dfe.NewMoneyFromCents(123456),
				ReceivableValue:     dfe.NewMoneyFromCents(120000),
				ICMSValue:           dfe.NewMoneyFromCents(14815),
				TotTribValue:        dfe.NewMoneyFromCents(2000),
				CargaValue:          dfe.NewMoneyFromCents(9990000),
				ProdutoPredominante: "Peças",
				NFeChaves:           []string{"35260922222222000122550010000000011000000011"},
				Situacao:            cte.SituacaoAutorizada,
				LayoutVersion:       "4.00",
				ParseWarnings:       []string{"aviso"},
			},
			RelationID:       "rel-1",
			CompanyRole:      cte.CompanyRoleTomador,
			Papeis:           []cte.CompanyRole{cte.CompanyRoleTomador, cte.CompanyRoleRemetente},
			VisibilityReason: cte.VisibilityReasonExactTomador,
			FirstSeenNSU:     &nsu,
			LastSeenNSU:      &nsu,
			EventCount:       2,
		},
		{
			Document: cte.Document{
				ID:            "doc-2",
				Modelo:        "67",
				TipoDocumento: cte.TipoDocumentoCTeOS,
				Situacao:      cte.SituacaoCancelada,
			},
			RelationID:  "rel-2",
			CompanyRole: cte.CompanyRoleNone,
		},
	}

	rows := CTeRows(docs)
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2", len(rows))
	}

	row := rows[0]
	if row.ID != "rel-1" || row.DocumentID != "doc-1" || row.ChaveAcesso != string(docs[0].ChaveAcesso) {
		t.Errorf("ID = %q, DocumentID = %q, ChaveAcesso = %q", row.ID, row.DocumentID, row.ChaveAcesso)
	}
	if row.TpAmb != "2" || row.Modelo != "57" || row.TipoDocumento != "cte" || row.TpCTe != "0" || row.TpServ != "2" || row.Modal != "01" {
		t.Errorf("codes = %q %q %q %q %q %q", row.TpAmb, row.Modelo, row.TipoDocumento, row.TpCTe, row.TpServ, row.Modal)
	}
	if row.TotalValue != 123456 || row.ReceivableValue != 120000 || row.ICMSValue != 14815 || row.TotTribValue != 2000 || row.CargaValue != 9990000 {
		t.Errorf("values = %d %d %d %d %d, want cents", row.TotalValue, row.ReceivableValue, row.ICMSValue, row.TotTribValue, row.CargaValue)
	}
	if row.MunIni != (CTeMunicipio{Codigo: "3550308", Nome: "São Paulo", UF: "SP"}) || row.MunFim.UF != "RJ" {
		t.Errorf("MunIni = %+v, MunFim = %+v", row.MunIni, row.MunFim)
	}
	if row.EmitenteCNPJ != "11111111000111" || row.RemetenteName != "Remetente" || row.DestinatarioCNPJ != "33333333000133" ||
		row.ExpedidorName != "Expedidor" || row.RecebedorCNPJ != "55555555000155" {
		t.Errorf("parties = %+v", row)
	}
	if row.TomadorCNPJ != "22222222000122" || row.TomadorName != "Remetente" || row.TomadorIE != "123" || row.TomadorUF != "SP" || row.TomadorIndicador != "0" {
		t.Errorf("tomador = %q %q %q %q %q", row.TomadorCNPJ, row.TomadorName, row.TomadorIE, row.TomadorUF, row.TomadorIndicador)
	}
	if row.Situacao != "autorizada" || row.CompanyRole != "tomador" || row.VisibilityReason != "exact_tomador" {
		t.Errorf("enums = %q %q %q", row.Situacao, row.CompanyRole, row.VisibilityReason)
	}
	if len(row.Papeis) != 2 || row.Papeis[0] != "tomador" || row.Papeis[1] != "remetente" {
		t.Errorf("Papeis = %v", row.Papeis)
	}
	if len(row.NFeChaves) != 1 || row.NFeChaves[0] != docs[0].NFeChaves[0] {
		t.Errorf("NFeChaves = %v", row.NFeChaves)
	}
	if row.AuthorizedAt == nil || !row.AuthorizedAt.Equal(authorized) || row.FirstSeenNSU == nil || *row.FirstSeenNSU != 42 {
		t.Errorf("AuthorizedAt = %v, FirstSeenNSU = %v", row.AuthorizedAt, row.FirstSeenNSU)
	}
	if row.EventCount != 2 || row.LayoutVersion != "4.00" || len(row.ParseWarnings) != 1 {
		t.Errorf("EventCount = %d, LayoutVersion = %q, ParseWarnings = %v", row.EventCount, row.LayoutVersion, row.ParseWarnings)
	}

	empty := rows[1]
	if empty.Modelo != "67" || empty.TipoDocumento != "cte_os" || empty.Situacao != "cancelada" || empty.CompanyRole != "none" {
		t.Errorf("enums = %q %q %q %q", empty.Modelo, empty.TipoDocumento, empty.Situacao, empty.CompanyRole)
	}
	if empty.AuthorizedAt != nil || empty.FirstSeenNSU != nil || empty.TomadorIndicador != "" {
		t.Errorf("AuthorizedAt = %v, FirstSeenNSU = %v, TomadorIndicador = %q", empty.AuthorizedAt, empty.FirstSeenNSU, empty.TomadorIndicador)
	}
	if empty.Papeis == nil || empty.NFeChaves == nil || empty.ParseWarnings == nil {
		t.Errorf("Papeis = %v, NFeChaves = %v, ParseWarnings = %v, want empty slices so the frontend gets []", empty.Papeis, empty.NFeChaves, empty.ParseWarnings)
	}
}

func TestCTeEvents(t *testing.T) {
	registeredAt := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	got := CTeEvents([]cte.Event{{
		ID:            "ev-1",
		TpEvento:      cte.TpEventoCCe,
		Type:          cte.EventTypeCartaCorrecao,
		NSeqEvento:    1,
		Description:   "Carta de Correção",
		RegisteredAt:  &registeredAt,
		Protocolo:     "p",
		CStat:         "135",
		XMotivo:       "Evento registrado",
		AutorCNPJ:     "11111111000111",
		Justificativa: "j",
		Observacao:    "o",
		Correcao:      "ide.natOp=Transporte",
		Registered:    true,
	}})
	want := CTeEvent{
		ID:            "ev-1",
		TpEvento:      "110110",
		Type:          "carta_correcao",
		NSeqEvento:    1,
		Description:   "Carta de Correção",
		RegisteredAt:  &registeredAt,
		Protocolo:     "p",
		CStat:         "135",
		XMotivo:       "Evento registrado",
		AutorCNPJ:     "11111111000111",
		Justificativa: "j",
		Observacao:    "o",
		Correcao:      "ide.natOp=Transporte",
		Registered:    true,
	}
	if len(got) != 1 || got[0] != want {
		t.Errorf("CTeEvents = %+v, want [%+v]", got, want)
	}
	if events := CTeEvents(nil); events == nil || len(events) != 0 {
		t.Errorf("CTeEvents(nil) = %v, want an empty slice", events)
	}
}

func TestCTeResetResultFrom(t *testing.T) {
	got := CTeResetResultFrom(app.CTeResetResult{
		CompanyName: "Empresa",
		CNPJ:        "12345678000195",
		Environment: "producao",
		ResetCounts: cte.ResetCounts{CompanyDocuments: 5, Documents: 3, Events: 7, ExportMarks: 2},
	})
	want := CTeResetResult{
		CompanyName:      "Empresa",
		CNPJ:             "12345678000195",
		CompanyDocuments: 5,
		Documents:        3,
		Events:           7,
		ExportMarks:      2,
	}
	if got != want {
		t.Errorf("CTeResetResultFrom = %+v, want %+v", got, want)
	}
}
