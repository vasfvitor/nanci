package godanfsev2

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Fixtures under testdata/ come from github.com/wevertonj/go-danfse-v2
// (BSD-3-Clause, v0.1.0). They are real-shaped NFS-e documents with the
// sped.fazenda.gov.br namespace, which the renderer requires.

// TestRenderFixtures renders every fixture through the adapter and checks that
// a PDF comes out. It guards against upstream renderer upgrades that start
// rejecting inputs the app already stores. Set DANFSE_PDF_OUT to a directory
// to keep the generated PDFs for visual inspection.
func TestRenderFixtures(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("testdata", "*.xml"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found")
	}

	outDir := os.Getenv("DANFSE_PDF_OUT")
	r := New()
	for _, path := range fixtures {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			xmlData, err := os.ReadFile(path) // #nosec G304 -- path comes from the testdata glob.
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			pdf := mustRender(t, r, xmlData)
			if outDir != "" {
				out := filepath.Join(outDir, name+".pdf")
				if err := os.WriteFile(out, pdf, 0o600); err != nil { // #nosec G703 -- opt-in debug output dir chosen by the developer.
					t.Fatalf("write %s: %v", out, err)
				}
			}
		})
	}
}

// TestRenderExtractsNFSeFromEnvelope checks that the adapter strips any
// wrapper around the <NFSe> element before handing it to the renderer.
func TestRenderExtractsNFSeFromEnvelope(t *testing.T) {
	xmlData, err := os.ReadFile(filepath.Join("testdata", "mock_danfse.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Drop the XML declaration so it can be nested inside another element.
	if i := bytes.Index(xmlData, []byte("?>")); i >= 0 {
		xmlData = xmlData[i+2:]
	}
	wrapped := append([]byte("<envelope><meta>x</meta>"), xmlData...)
	wrapped = append(wrapped, []byte("</envelope>")...)

	mustRender(t, New(), wrapped)
}

func mustRender(t *testing.T, r interface {
	Render([]byte) ([]byte, error)
}, xmlData []byte) []byte {
	t.Helper()
	pdf, err := r.Render(xmlData)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("expected PDF output, got %q...", pdf[:min(len(pdf), 16)])
	}
	return pdf
}
