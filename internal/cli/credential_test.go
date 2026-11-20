package cli

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
)

// TestTerminalCredentialProvider_NoTerminal asserts that without a saved
// password, NANCI_CERT_PASSWORD or a terminal (a script or an AI agent), the
// provider fails with what to do and writes no prompt nobody would answer.
func TestTerminalCredentialProvider_NoTerminal(t *testing.T) {
	t.Setenv("NANCI_CERT_PASSWORD", "")
	dir := t.TempDir()
	in, err := os.CreateTemp(dir, "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.CreateTemp(dir, "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = out.Close() }()

	_, err = TerminalCredentialProvider{In: in, Out: out}.GetCertPassword(context.Background(), app.CertPasswordRequest{})
	if !errors.Is(err, errNoPasswordTerminal) {
		t.Fatalf("err = %v, want errNoPasswordTerminal", err)
	}
	if code := app.ErrorCode(err); code != app.ErrorCodePasswordRequired {
		t.Errorf("ErrorCode = %q, want %q", code, app.ErrorCodePasswordRequired)
	}
	if info, statErr := out.Stat(); statErr != nil || info.Size() != 0 {
		t.Errorf("prompt written without a terminal (size %d, err %v)", info.Size(), statErr)
	}

	t.Setenv("NANCI_CERT_PASSWORD", "segredo")
	pass, err := TerminalCredentialProvider{In: in, Out: out}.GetCertPassword(context.Background(), app.CertPasswordRequest{})
	if err != nil || string(pass) != "segredo" {
		t.Errorf("with NANCI_CERT_PASSWORD: pass = %q, err = %v", pass, err)
	}
}
