package desktopapi

import (
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func TestCTeRows(t *testing.T) {
	authorized := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	viewed := authorized.Add(2 * time.Hour)
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
			ViewedAt:         &viewed,
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
	if row.ViewedAt == nil || !row.ViewedAt.Equal(viewed) {
		t.Errorf("ViewedAt = %v, want %v", row.ViewedAt, viewed)
	}

	empty := rows[1]
	if empty.Modelo != "67" || empty.TipoDocumento != "cte_os" || empty.Situacao != "cancelada" || empty.CompanyRole != "none" {
		t.Errorf("enums = %q %q %q %q", empty.Modelo, empty.TipoDocumento, empty.Situacao, empty.CompanyRole)
	}
	if empty.AuthorizedAt != nil || empty.FirstSeenNSU != nil || empty.ViewedAt != nil || empty.TomadorIndicador != "" {
		t.Errorf("AuthorizedAt = %v, FirstSeenNSU = %v, ViewedAt = %v, TomadorIndicador = %q", empty.AuthorizedAt, empty.FirstSeenNSU, empty.ViewedAt, empty.TomadorIndicador)
	}
	if empty.Papeis == nil || empty.NFeChaves == nil || empty.ParseWarnings == nil {
		t.Errorf("Papeis = %v, NFeChaves = %v, ParseWarnings = %v, want empty slices so the frontend gets []", empty.Papeis, empty.NFeChaves, empty.ParseWarnings)
	}
}

func TestCTeRowHasDACTE(t *testing.T) {
	tests := []struct {
		tipo cte.TipoDocumento
		want bool
	}{
		{cte.TipoDocumentoCTe, true},
		{cte.TipoDocumentoCTeOS, false},
		{cte.TipoDocumentoGTVe, false},
		{cte.TipoDocumentoCTeSimplificado, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.tipo), func(t *testing.T) {
			rows := CTeRows([]cte.CompanyDocument{{Document: cte.Document{TipoDocumento: tt.tipo}}})
			if rows[0].HasDACTE != tt.want {
				t.Errorf("HasDACTE = %v, want %v", rows[0].HasDACTE, tt.want)
			}
		})
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

func TestCTeExportResultFrom(t *testing.T) {
	got := CTeExportResultFrom(app.CTeExportResult{
		ExportResult:    app.ExportResult{OutPath: "dactes.zip", Format: "dacte", Incremental: true, ExportedCount: 3},
		SkippedSemDACTE: 2,
	})
	want := CTeExportResult{
		ExportResult:    ExportResult{OutPath: "dactes.zip", Format: "dacte", Incremental: true, ExportedCount: 3},
		SkippedSemDACTE: 2,
	}
	if got != want {
		t.Errorf("CTeExportResultFrom = %+v, want %+v", got, want)
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
