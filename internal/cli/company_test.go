package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempDataDir isolates each CLI test in its own fresh SQLite database by
// pointing NANCI_DATA_DIR at a per-test temp directory. This is a transitional
// isolation mechanism; the structural plan will later replace newApp() with an
// injectable AppFactory so tests can avoid the filesystem entirely.
func withTempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "blobs"), 0o750); err != nil {
		t.Fatalf("create temp data dir: %v", err)
	}
	t.Setenv("NANCI_DATA_DIR", dir)
	return dir
}

// TestCompanyList_EmptyDB_PrintsNoCompanies asserts the company list subcommand
// routes success output through cmd.OutOrStdout and reports an empty database
// without printing usage or touching stderr.
func TestCompanyList_EmptyDB_PrintsNoCompanies(t *testing.T) {
	resetRoot(t)
	withTempDataDir(t)

	rootCmd.SetArgs([]string{"company", "list"})
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	if err := Execute(context.Background()); err != nil {
		t.Fatalf("Execute returned %v, want nil", err)
	}
	if got, want := strings.TrimSpace(out.String()), "Nenhuma empresa cadastrada."; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty: %q", errOut.String())
	}
}

// TestCompanyAdd_InvalidCNPJ_ReturnsErrorWithoutUsage asserts that a validation
// failure inside the app layer surfaces as the sole error on Execute's return
// value, with neither usage text nor the error echoed by Cobra to stderr.
func TestCompanyAdd_InvalidCNPJ_ReturnsErrorWithoutUsage(t *testing.T) {
	resetRoot(t)
	withTempDataDir(t)

	rootCmd.SetArgs([]string{
		"company", "add",
		"--cnpj", "00000000000000",
		"--name", "Acme LTDA",
		"--cert", "/nonexistent/cert.pfx",
		"--env", "producao_restrita",
	})
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	err := Execute(context.Background())
	if err == nil {
		t.Fatal("Execute returned nil for invalid CNPJ, want error")
	}
	// Cobra must stay silent: no usage text on either stream.
	if strings.Contains(out.String(), "Usage") || strings.Contains(out.String(), "Flags") {
		t.Errorf("stdout leaked usage text: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty (expected silence): %q", errOut.String())
	}
	if !strings.Contains(err.Error(), "CNPJ") && !strings.Contains(err.Error(), "cnpj") {
		t.Errorf("error %q does not mention CNPJ", err.Error())
	}
}

// TestCompanyList_MissingRequiredFlag asserts missing required flags surface as
// the single returned error without Cobra echoing usage to stderr. This covers
// the SilenceErrors path for flag-validation failures specifically.
func TestCompanyList_MissingRequiredFlag(t *testing.T) {
	resetRoot(t)
	withTempDataDir(t)

	// list requires --cnpj; omit it.
	rootCmd.SetArgs([]string{"list"})
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	err := Execute(context.Background())
	if err == nil {
		t.Fatal("Execute returned nil for missing required flag, want error")
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty: %q", errOut.String())
	}
}
