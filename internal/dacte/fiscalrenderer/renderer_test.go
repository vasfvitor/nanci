package fiscalrenderer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/dacte"
)

// TestRenderProcCTe renders the procCTe fixture of internal/cte, with and
// without the cancellation watermark. It guards against library upgrades
// that start rejecting the XML nanci stores.
func TestRenderProcCTe(t *testing.T) {
	xmlData, err := os.ReadFile(filepath.Join("..", "..", "cte", "testdata", "procte.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for _, opts := range []dacte.Options{{}, {Cancelada: true}} {
		pdf, err := New().Render(xmlData, opts)
		if err != nil {
			t.Fatalf("Render(%+v): %v", opts, err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
			t.Fatalf("Render(%+v) = %q..., want a PDF", opts, pdf[:min(len(pdf), 16)])
		}
	}
}
