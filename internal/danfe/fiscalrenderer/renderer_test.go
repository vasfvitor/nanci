package fiscalrenderer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/danfe"
)

// TestRenderProcNFe renders the procNFe fixture of internal/nfe, with and
// without the cancellation watermark. It guards against library upgrades
// that start rejecting the XML nanci stores.
func TestRenderProcNFe(t *testing.T) {
	xmlData, err := os.ReadFile(filepath.Join("..", "..", "nfe", "testdata", "procnfe.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for _, opts := range []danfe.Options{{}, {Cancelada: true}} {
		pdf, err := New().Render(xmlData, opts)
		if err != nil {
			t.Fatalf("Render(%+v): %v", opts, err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
			t.Fatalf("Render(%+v) = %q..., want a PDF", opts, pdf[:min(len(pdf), 16)])
		}
	}
}
