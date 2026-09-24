package cte

import "testing"

func TestMergeDocument(t *testing.T) {
	existing := Document{
		ID:        "existing-id",
		Emitente:  Party{Name: "OLD NAME"},
		Protocolo: "135260000000001",
		Situacao:  SituacaoAutorizada,
		RawHash:   "hash-old",
	}
	incoming := Document{
		ID:        "incoming-id",
		Emitente:  Party{Name: "NEW NAME"},
		Protocolo: "135260000000001",
		Situacao:  SituacaoAutorizada,
		RawHash:   "hash-new",
	}
	with := func(d Document, change func(*Document)) Document {
		change(&d)
		return d
	}

	tests := []struct {
		name         string
		existing     Document
		incoming     Document
		wantSituacao Situacao
	}{
		{"newer replaces", existing, incoming, SituacaoAutorizada},
		{
			"cancelada existing wins over autorizada incoming",
			with(existing, func(d *Document) { d.Situacao = SituacaoCancelada }),
			incoming,
			SituacaoCancelada,
		},
		{
			"cancelada incoming wins over autorizada",
			existing,
			with(incoming, func(d *Document) { d.Situacao = SituacaoCancelada }),
			SituacaoCancelada,
		},
		{
			"cancelada beats denegada",
			with(existing, func(d *Document) { d.Situacao = SituacaoDenegada }),
			with(incoming, func(d *Document) { d.Situacao = SituacaoCancelada }),
			SituacaoCancelada,
		},
		{
			"denegada is kept over autorizada",
			with(existing, func(d *Document) { d.Situacao = SituacaoDenegada }),
			incoming,
			SituacaoDenegada,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeDocument(tt.existing, tt.incoming)
			if got.ID != "existing-id" {
				t.Errorf("ID = %q, want the existing ID", got.ID)
			}
			if got.Emitente.Name != "NEW NAME" || got.RawHash != "hash-new" {
				t.Errorf("(name, raw hash) = (%q, %q), want the incoming values", got.Emitente.Name, got.RawHash)
			}
			if got.Situacao != tt.wantSituacao {
				t.Errorf("Situacao = %q, want %q", got.Situacao, tt.wantSituacao)
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
		{
			"other events do not change it",
			SituacaoAutorizada,
			[]EventType{EventTypeCartaCorrecao, EventTypeComprovanteEntrega, EventTypeMDFeAutorizado, EventTypePrestacaoDesacordo},
			SituacaoAutorizada,
		},
		{"cancelamento cancels", SituacaoAutorizada, []EventType{EventTypeCartaCorrecao, EventTypeCancelamento}, SituacaoCancelada},
		{
			"cancelamento of a comprovante does not cancel the CT-e",
			SituacaoAutorizada,
			[]EventType{EventTypeComprovanteEntrega, EventTypeCancelamentoComprovanteEntrega},
			SituacaoAutorizada,
		},
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
