package cte

import (
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
)

func TestEventTypeFromTpEvento(t *testing.T) {
	tests := map[string]EventType{
		TpEventoCCe:                    EventTypeCartaCorrecao,
		TpEventoCancelamento:           EventTypeCancelamento,
		TpEventoEPEC:                   EventTypeEPEC,
		TpEventoRegistroMultimodal:     EventTypeRegistroMultimodal,
		TpEventoGTV:                    EventTypeGTV,
		TpEventoComprovanteEntrega:     EventTypeComprovanteEntrega,
		TpEventoCancComprovanteEntrega: EventTypeCancelamentoComprovanteEntrega,
		TpEventoInsucessoEntrega:       EventTypeInsucessoEntrega,
		TpEventoCancInsucessoEntrega:   EventTypeCancelamentoInsucessoEntrega,
		TpEventoPrestacaoDesacordo:     EventTypePrestacaoDesacordo,
		TpEventoCancPrestacaoDesacordo: EventTypeCancelamentoDesacordo,
		TpEventoMDFeAutorizado:         EventTypeMDFeAutorizado,
		TpEventoMDFeCancelado:          EventTypeMDFeCancelado,
		"210210":                       EventTypeUnknown, // ciência da operação, NF-e only
		"":                             EventTypeUnknown,
	}
	for tp, want := range tests {
		got := EventTypeFromTpEvento(tp)
		if got != want {
			t.Errorf("EventTypeFromTpEvento(%q) = %q, want %q", tp, got, want)
		}
		if !got.Valid() {
			t.Errorf("EventTypeFromTpEvento(%q) = %q is not Valid", tp, got)
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
		{"company role", func(v string) error { _, err := ParseCompanyRole(v); return err }, "expedidor"},
		{"tipo documento", func(v string) error { _, err := ParseTipoDocumento(v); return err }, "cte_simplificado"},
	}
	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.parse(tt.value); err != nil {
				t.Errorf("parse(%q): %v", tt.value, err)
			}
			if err := tt.parse("bogus"); !errors.Is(err, dfe.ErrInvalidEnum) {
				t.Errorf("parse(bogus) error = %v, want ErrInvalidEnum", err)
			}
		})
	}
	if !VisibilityReasonExactRecebedor.Valid() || VisibilityReason("exact_transportador").Valid() {
		t.Error("VisibilityReason.Valid does not match the CT-e reasons")
	}
}

func TestTipoDocumentoModelo(t *testing.T) {
	tests := map[TipoDocumento]string{
		TipoDocumentoCTe:             "57",
		TipoDocumentoCTeSimplificado: "57",
		TipoDocumentoGTVe:            "64",
		TipoDocumentoCTeOS:           "67",
		"bogus":                      "",
	}
	for tipo, want := range tests {
		if got := tipo.Modelo(); got != want {
			t.Errorf("%q.Modelo() = %q, want %q", tipo, got, want)
		}
	}
}

func TestClassifySchema(t *testing.T) {
	tests := map[string]SchemaKind{
		"procCTe_v4.00.xsd":       SchemaProcCTe,
		"procCTe_v3.00.xsd":       SchemaProcCTe,
		"procCTeOS_v4.00.xsd":     SchemaProcCTeOS,
		"procGTVe_v4.00.xsd":      SchemaProcGTVe,
		"procCTeSimp_v4.00.xsd":   SchemaProcCTeSimp,
		"procEventoCTe_v4.00.xsd": SchemaProcEventoCTe,
		"procCTe_v4.00":           SchemaProcCTe,
		"procNFe_v4.00.xsd":       SchemaUnknown,
		"procMDFe_v3.00.xsd":      SchemaUnknown,
		"":                        SchemaUnknown,
	}
	for schema, want := range tests {
		if got := ClassifySchema(schema); got != want {
			t.Errorf("ClassifySchema(%q) = %s, want %s", schema, got, want)
		}
	}
}
