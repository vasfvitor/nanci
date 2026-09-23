package nfe

import (
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestEventTypeFromTpEvento(t *testing.T) {
	tests := map[string]EventType{
		TpEventoCCe:              EventTypeCartaCorrecao,
		TpEventoCancelamento:     EventTypeCancelamento,
		TpEventoCancSubstituicao: EventTypeCancelamento,
		TpEventoConfirmacao:      EventTypeConfirmacao,
		TpEventoCiencia:          EventTypeCiencia,
		TpEventoDesconhecimento:  EventTypeDesconhecimento,
		TpEventoNaoRealizada:     EventTypeNaoRealizada,
		"610600":                 EventTypeUnknown, // registro de CT-e
		"":                       EventTypeUnknown,
	}
	for tp, want := range tests {
		if got := EventTypeFromTpEvento(tp); got != want {
			t.Errorf("EventTypeFromTpEvento(%q) = %q, want %q", tp, got, want)
		}
	}
}

func TestEnumParsers(t *testing.T) {
	valid := []struct {
		name  string
		parse func(string) error
		value string
	}{
		{"situacao", func(v string) error { _, err := ParseSituacao(v); return err }, "cancelada"},
		{"completeness", func(v string) error { _, err := ParseCompleteness(v); return err }, "resumo"},
		{"company role", func(v string) error { _, err := ParseCompanyRole(v); return err }, "transportador"},
		{"manifestacao", func(v string) error { _, err := ParseManifestacao(v); return err }, "nao_realizada"},
	}
	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.parse(tt.value); err != nil {
				t.Errorf("parse(%q): %v", tt.value, err)
			}
			if err := tt.parse("bogus"); !errors.Is(err, nfse.ErrInvalidEnum) {
				t.Errorf("parse(bogus) error = %v, want ErrInvalidEnum", err)
			}
		})
	}
}

func TestClassifySchema(t *testing.T) {
	tests := map[string]SchemaKind{
		"resNFe_v1.01.xsd":        SchemaResNFe,
		"procNFe_v4.00.xsd":       SchemaProcNFe,
		"resEvento_v1.01.xsd":     SchemaResEvento,
		"procEventoNFe_v1.00.xsd": SchemaProcEventoNFe,
		"resNFe_v1.01":            SchemaResNFe,
		"procCTe_v4.00.xsd":       SchemaUnknown,
		"resNFeX_v1.01.xsd":       SchemaUnknown,
		"":                        SchemaUnknown,
	}
	for schema, want := range tests {
		if got := ClassifySchema(schema); got != want {
			t.Errorf("ClassifySchema(%q) = %s, want %s", schema, got, want)
		}
	}
}
