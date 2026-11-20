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

// TestReadPasswordContext_Cancel checks that Ctrl+C at the prompt (a
// cancelled ctx) ends the read with ErrOperationCanceled and restores the
// console, while the read itself is still blocked.
func TestReadPasswordContext_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	blocked := make(chan struct{})
	defer close(blocked)
	restored := false
	go cancel()
	pass, err := readPasswordContext(ctx, func() ([]byte, error) {
		<-blocked
		return nil, nil
	}, func() { restored = true })
	if !errors.Is(err, app.ErrOperationCanceled) || pass != nil {
		t.Errorf("pass = %q, err = %v; want ErrOperationCanceled", pass, err)
	}
	if !restored {
		t.Error("console not restored on cancel")
	}
	if code := app.ErrorCode(err); code != app.ErrorCodeCanceled {
		t.Errorf("code = %q, want %q", code, app.ErrorCodeCanceled)
	}
}

func TestReadPasswordContext_Read(t *testing.T) {
	restored := false
	pass, err := readPasswordContext(context.Background(), func() ([]byte, error) {
		return []byte("segredo"), nil
	}, func() { restored = true })
	if err != nil || string(pass) != "segredo" {
		t.Errorf("pass = %q, err = %v", pass, err)
	}
	if restored {
		t.Error("restore called without a cancel: term.ReadPassword restores by itself")
	}
}
