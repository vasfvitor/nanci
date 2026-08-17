package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

const mockCertPassword = "mockdata"

func mockCertPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "foundation", "cert", "testdata", "cert_a1_mock_70860312000150.pfx")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("mock cert not found: %v", err)
	}
	return path
}

type fallbackStub struct {
	pass  string
	calls int
}

func (s *fallbackStub) GetCertPassword(context.Context, CertPasswordRequest) (string, error) {
	s.calls++
	return s.pass, nil
}

// stubKeyring replaces the keyring seams for the duration of the test.
func stubKeyring(t *testing.T, get func(string, string) (string, error), set func(string, string, string) error) {
	t.Helper()
	origGet, origSet := keyringGet, keyringSet
	keyringGet, keyringSet = get, set
	t.Cleanup(func() { keyringGet, keyringSet = origGet, origSet })
}

func TestKeyringCredentialProvider_UsesStoredPassword(t *testing.T) {
	certPath := mockCertPath(t)
	setCalls := 0
	stubKeyring(t,
		func(string, string) (string, error) { return mockCertPassword, nil },
		func(string, string, string) error { setCalls++; return nil },
	)
	fallback := &fallbackStub{pass: "unused"}

	pass, err := KeyringCredentialProvider{Fallback: fallback}.GetCertPassword(context.Background(), CertPasswordRequest{CredentialID: "cred-1", CertPath: certPath})
	if err != nil {
		t.Fatalf("GetCertPassword: %v", err)
	}
	if pass != mockCertPassword {
		t.Fatalf("pass = %q, want %q", pass, mockCertPassword)
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback called %d times, want 0", fallback.calls)
	}
	if setCalls != 0 {
		t.Fatalf("keyring.Set called %d times, want 0", setCalls)
	}
}

func TestKeyringCredentialProvider_FallsBackWhenStoredPasswordInvalid(t *testing.T) {
	certPath := mockCertPath(t)
	var saved string
	stubKeyring(t,
		func(string, string) (string, error) { return "stale", nil },
		func(_, _, pass string) error { saved = pass; return nil },
	)
	fallback := &fallbackStub{pass: mockCertPassword}
	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pass, err := KeyringCredentialProvider{Fallback: fallback, Log: log}.GetCertPassword(context.Background(), CertPasswordRequest{CredentialID: "cred-1", CertPath: certPath})
	if err != nil {
		t.Fatalf("GetCertPassword: %v", err)
	}
	if pass != mockCertPassword || fallback.calls != 1 {
		t.Fatalf("pass = %q, fallback calls = %d; want %q and 1", pass, fallback.calls, mockCertPassword)
	}
	if saved != mockCertPassword {
		t.Fatalf("saved %q to keyring, want %q", saved, mockCertPassword)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("não abre o certificado")) {
		t.Fatalf("expected debug log about stale keyring password, got:\n%s", logBuf.String())
	}
}

func TestKeyringCredentialProvider_WarnsWhenSaveFails(t *testing.T) {
	certPath := mockCertPath(t)
	stubKeyring(t,
		func(string, string) (string, error) { return "", keyring.ErrNotFound },
		func(string, string, string) error { return errors.New("dbus unavailable") },
	)
	fallback := &fallbackStub{pass: mockCertPassword}
	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	pass, err := KeyringCredentialProvider{Fallback: fallback, Log: log}.GetCertPassword(context.Background(), CertPasswordRequest{CredentialID: "cred-1", CertPath: certPath})
	if err != nil {
		t.Fatalf("GetCertPassword: %v", err)
	}
	if pass != mockCertPassword {
		t.Fatalf("pass = %q, want %q", pass, mockCertPassword)
	}
	out := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("level=WARN")) || !bytes.Contains(logBuf.Bytes(), []byte("dbus unavailable")) {
		t.Fatalf("expected WARN log with keyring error, got:\n%s", out)
	}
}

func TestKeyringCredentialProvider_RejectsInvalidFallbackPassword(t *testing.T) {
	certPath := mockCertPath(t)
	setCalls := 0
	stubKeyring(t,
		func(string, string) (string, error) { return "", keyring.ErrNotFound },
		func(string, string, string) error { setCalls++; return nil },
	)
	fallback := &fallbackStub{pass: "wrong"}

	_, err := KeyringCredentialProvider{Fallback: fallback}.GetCertPassword(context.Background(), CertPasswordRequest{CredentialID: "cred-1", CertPath: certPath})
	if err == nil {
		t.Fatal("expected error for invalid password")
	}
	if setCalls != 0 {
		t.Fatalf("keyring.Set called %d times for an invalid password, want 0", setCalls)
	}
}
