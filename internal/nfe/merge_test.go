package nfe

import (
	"testing"
	"time"
)

func TestMergeDocument(t *testing.T) {
	authorized := time.Date(2026, 9, 1, 9, 15, 42, 0, time.UTC)

	resumo := Document{
		ID:           "existing-id",
		ChaveAcesso:  keyProcNFe,
		EmitenteName: "RESUMO NAME",
		AuthorizedAt: &authorized,
		Protocolo:    "135260000000001",
		Situacao:     SituacaoAutorizada,
		Completeness: CompletenessResumo,
		RawHash:      "hash-resumo",
	}
	completa := Document{
		ID:               "incoming-id",
		ChaveAcesso:      keyProcNFe,
		EmitenteName:     "COMPLETA NAME",
		DestinatarioCNPJ: cnpjMock,
		AuthorizedAt:     &authorized,
		Protocolo:        "135260000000001",
		Situacao:         SituacaoAutorizada,
		Completeness:     CompletenessCompleta,
		RawHash:          "hash-completa",
	}
	with := func(d Document, change func(*Document)) Document {
		change(&d)
		return d
	}

	tests := []struct {
		name             string
		existing         Document
		incoming         Document
		wantCompleteness Completeness
		wantName         string
		wantSituacao     Situacao
		wantRawHash      string
		wantResumoHash   string
		wantProtocolo    string
	}{
		{
			name:             "resumo upgraded to completa",
			existing:         resumo,
			incoming:         completa,
			wantCompleteness: CompletenessCompleta,
			wantName:         "COMPLETA NAME",
			wantSituacao:     SituacaoAutorizada,
			wantRawHash:      "hash-completa",
			wantResumoHash:   "hash-resumo",
			wantProtocolo:    "135260000000001",
		},
		{
			name:             "completa never downgrades to resumo",
			existing:         with(completa, func(d *Document) { d.ID = "existing-id" }),
			incoming:         with(resumo, func(d *Document) { d.ID = "incoming-id" }),
			wantCompleteness: CompletenessCompleta,
			wantName:         "COMPLETA NAME",
			wantSituacao:     SituacaoAutorizada,
			wantRawHash:      "hash-completa",
			wantResumoHash:   "hash-resumo",
			wantProtocolo:    "135260000000001",
		},
		{
			name:             "cancelled resumo over completa raises situacao only",
			existing:         with(completa, func(d *Document) { d.ID = "existing-id" }),
			incoming:         with(resumo, func(d *Document) { d.Situacao = SituacaoCancelada }),
			wantCompleteness: CompletenessCompleta,
			wantName:         "COMPLETA NAME",
			wantSituacao:     SituacaoCancelada,
			wantRawHash:      "hash-completa",
			wantResumoHash:   "hash-resumo",
			wantProtocolo:    "135260000000001",
		},
		{
			name:             "resumo fills a missing protocolo on completa",
			existing:         with(completa, func(d *Document) { d.ID = "existing-id"; d.Protocolo = "" }),
			incoming:         resumo,
			wantCompleteness: CompletenessCompleta,
			wantName:         "COMPLETA NAME",
			wantSituacao:     SituacaoAutorizada,
			wantRawHash:      "hash-completa",
			wantResumoHash:   "hash-resumo",
			wantProtocolo:    "135260000000001",
		},
		{
			name:             "cancelada existing wins over autorizada completa",
			existing:         with(resumo, func(d *Document) { d.Situacao = SituacaoCancelada }),
			incoming:         completa,
			wantCompleteness: CompletenessCompleta,
			wantName:         "COMPLETA NAME",
			wantSituacao:     SituacaoCancelada,
			wantRawHash:      "hash-completa",
			wantResumoHash:   "hash-resumo",
			wantProtocolo:    "135260000000001",
		},
		{
			name:             "cancelada beats denegada",
			existing:         with(resumo, func(d *Document) { d.Situacao = SituacaoDenegada }),
			incoming:         with(resumo, func(d *Document) { d.Situacao = SituacaoCancelada; d.RawHash = "hash-resumo-2" }),
			wantCompleteness: CompletenessResumo,
			wantName:         "RESUMO NAME",
			wantSituacao:     SituacaoCancelada,
			wantRawHash:      "hash-resumo-2",
			wantResumoHash:   "",
			wantProtocolo:    "135260000000001",
		},
		{
			name:             "completa over completa keeps the resumo hash",
			existing:         with(completa, func(d *Document) { d.ID = "existing-id"; d.ResumoRawHash = "hash-resumo" }),
			incoming:         with(completa, func(d *Document) { d.RawHash = "hash-completa-2"; d.Situacao = SituacaoDenegada }),
			wantCompleteness: CompletenessCompleta,
			wantName:         "COMPLETA NAME",
			wantSituacao:     SituacaoDenegada,
			wantRawHash:      "hash-completa-2",
			wantResumoHash:   "hash-resumo",
			wantProtocolo:    "135260000000001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeDocument(tt.existing, tt.incoming)
			if got.ID != "existing-id" {
				t.Errorf("ID = %q, want the existing ID", got.ID)
			}
			if got.Completeness != tt.wantCompleteness {
				t.Errorf("Completeness = %q, want %q", got.Completeness, tt.wantCompleteness)
			}
			if got.EmitenteName != tt.wantName {
				t.Errorf("EmitenteName = %q, want %q", got.EmitenteName, tt.wantName)
			}
			if got.Situacao != tt.wantSituacao {
				t.Errorf("Situacao = %q, want %q", got.Situacao, tt.wantSituacao)
			}
			if got.RawHash != tt.wantRawHash {
				t.Errorf("RawHash = %q, want %q", got.RawHash, tt.wantRawHash)
			}
			if got.ResumoRawHash != tt.wantResumoHash {
				t.Errorf("ResumoRawHash = %q, want %q", got.ResumoRawHash, tt.wantResumoHash)
			}
			if got.Protocolo != tt.wantProtocolo {
				t.Errorf("Protocolo = %q, want %q", got.Protocolo, tt.wantProtocolo)
			}
		})
	}
}

func TestSituacaoFromEvents(t *testing.T) {
	tests := []struct {
		name   string
		base   Situacao
		events []EventType
		want   Situacao
	}{
		{"no events", SituacaoAutorizada, nil, SituacaoAutorizada},
		{"manifestações do not change it", SituacaoAutorizada, []EventType{EventTypeCiencia, EventTypeConfirmacao, EventTypeCartaCorrecao}, SituacaoAutorizada},
		{"cancelamento cancels", SituacaoAutorizada, []EventType{EventTypeCiencia, EventTypeCancelamento}, SituacaoCancelada},
		{"denegada stays denegada", SituacaoDenegada, []EventType{EventTypeUnknown}, SituacaoDenegada},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SituacaoFromEvents(tt.base, tt.events); got != tt.want {
				t.Errorf("SituacaoFromEvents = %q, want %q", got, tt.want)
			}
		})
	}
}
