//go:build xmlsec

package sefaz

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSignEvento_Xmlsec1 checks the signature with a general XMLDSig
// implementation. Run with: go test -tags xmlsec ./internal/sefaz/
func TestSignEvento_Xmlsec1(t *testing.T) {
	xmlsec, err := exec.LookPath("xmlsec1")
	if err != nil {
		t.Skip("xmlsec1 is not installed")
	}

	signed, err := newMockSigner(t).SignEvento(cienciaEvento(), TpAmbProducao)
	if err != nil {
		t.Fatalf("SignEvento: %v", err)
	}
	path := filepath.Join(t.TempDir(), "evento.xml")
	if err := os.WriteFile(path, signed, 0o600); err != nil {
		t.Fatal(err)
	}

	// --insecure skips the certificate chain: the mock certificate is not
	// issued by a trusted CA, and only the signature is under test.
	cmd := exec.CommandContext(context.Background(), xmlsec, "--verify", "--insecure", "--enabled-key-data", "x509",
		"--id-attr:Id", nfeNamespace+":infEvento", path) // #nosec G204 -- fixed arguments.
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("xmlsec1 --verify: %v\n%s", err, out)
	}
}
