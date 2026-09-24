package sync

import (
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// The preview masks the whole document before the cut, so an identifier
// that straddles the 400-character limit is not left in clear.
func TestXMLPreviewMasksNFSeIdentifiersBeforeTheCut(t *testing.T) {
	prefix := "<NFSe><infNFSe><xLocEmi>" + strings.Repeat("x", 345) + "</xLocEmi>"
	data := prefix + "<prest><CNPJ>11222333000181</CNPJ></prest><toma><xNome>TOMADORA FICTICIA</xNome></toma></infNFSe></NFSe>"

	got := xmlPreview([]byte(data))
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Fatalf("preview = %q, want it truncated", got)
	}
	// The cut falls inside the CNPJ text: its first eight characters remain.
	if strings.Contains(got, "11222333") {
		t.Errorf("preview leaks the start of the CNPJ: %s", got)
	}
	if !strings.Contains(got, "<CNPJ>11******...(truncated)") {
		t.Errorf("preview lacks the masked CNPJ: %s", got)
	}
}

func TestProcessingErrorReportsSourceAttrs(t *testing.T) {
	err := &ProcessingError{
		Op:     "parse document",
		NSU:    7,
		Schema: "NFSe",
		Attrs:  []slog.Attr{slog.String("tipo_documento", "NFSE"), slog.String("tipo_evento", "CANCELAMENTO")},
		Err:    errors.New("bad xml"),
	}
	const want = "parse document failed (nsu=7, schema=NFSe, tipo_documento=NFSE, tipo_evento=CANCELAMENTO): bad xml"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	group := err.LogValue().Group()
	if last := group[len(group)-1]; last.Key != "tipo_evento" || last.Value.String() != "CANCELAMENTO" {
		t.Errorf("LogValue ends with %v, want tipo_evento=CANCELAMENTO", last)
	}
}
