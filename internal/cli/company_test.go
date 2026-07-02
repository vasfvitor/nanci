package cli

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/store"
)

// newInMemTestRoot builds a fresh root wired to an in-memory SQLite-backed
// app.App. Each call produces an isolated App, so tests are independent
// without touching the filesystem or shared package globals.
//
// The harness is white-box (package cli) so it can reach the package-private
// `rootCmd` etc. and use the AppFactory seam. The DB connection and blob
// store are scoped to the test via t.Cleanup.
func newInMemTestRoot(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	ctx := context.Background()
	db, err := store.OpenDB(ctx, "file::memory:?cache=shared", true)
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	docRepo := store.NewDocumentRepository(db)
	application, err := app.New(app.Dependencies{
		Log:                slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		CompanyRepo:        store.NewCompanyRepository(db),
		CredentialRepo:     store.NewCredentialRepository(db),
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     docRepo,
		DocumentTracker:    docRepo,
		XMLStore:           files.NewBlobStore(t.TempDir()),
		DataDir:            t.TempDir(),
		CredentialProvider: app.KeyringCredentialProvider{Fallback: TerminalCredentialProvider{In: os.Stdin, Out: os.Stderr}},
		DANFSeRenderer:     nil,
	})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	factoryCalled := false
	factory := func(ctx context.Context) (*app.App, func(), error) {
		if factoryCalled {
			return application, func() {}, nil
		}
		factoryCalled = true
		return application, func() {}, nil
	}

	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	v, tr := false, false
	root := NewRootCommand(CommandEnv{
		In:         os.Stdin,
		Out:        os.Stderr,
		Stdout:     out,
		AppFactory: factory,
		Verbose:    &v,
		Trace:      &tr,
	})
	return root, out, errOut
}

// TestCompanyList_EmptyDB_PrintsNoCompanies asserts the company list subcommand
// routes success output through cmd.OutOrStdout and reports an empty database
// without printing usage or touching stderr.
func TestCompanyList_EmptyDB_PrintsNoCompanies(t *testing.T) {
	root, out, errOut := newInMemTestRoot(t)

	root.SetArgs([]string{"company", "list"})
	root.SetErr(errOut)

	if err := root.ExecuteContext(context.Background()); err != nil {
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
	root, out, errOut := newInMemTestRoot(t)

	root.SetArgs([]string{
		"company", "add",
		"--cnpj", "00000000000000",
		"--name", "Acme LTDA",
		"--cert", "/nonexistent/cert.pfx",
		"--env", "producao_restrita",
	})
	root.SetErr(errOut)

	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("Execute returned nil for invalid CNPJ, want error")
	}
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

// TestList_MissingRequiredFlag asserts missing required flags surface as
// the single returned error without Cobra echoing usage to stderr.
func TestList_MissingRequiredFlag(t *testing.T) {
	root, _, errOut := newInMemTestRoot(t)

	// list requires --cnpj; omit it.
	root.SetArgs([]string{"list"})
	root.SetErr(errOut)

	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("Execute returned nil for missing required flag, want error")
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty: %q", errOut.String())
	}
}
