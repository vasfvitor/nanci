package cli

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
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
		Log:             slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		CompanyStore:    company.NewStore(db),
		CredentialStore: credential.NewStore(db),
		SyncRepo:        sync.NewStore(db),
		DocumentRepo:    docRepo,
		NFeRepo:         store.NewNFeRepository(db),
		CTeRepo:         store.NewCTeRepository(db),

		XMLStore:           files.NewBlobStore(t.TempDir()),
		DataDir:            t.TempDir(),
		CredentialProvider: app.KeyringCredentialProvider{Fallback: TerminalCredentialProvider{In: os.Stdin, Out: os.Stderr}},
		DANFSeRenderer:     nil,
	})
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	root, out := newTestRootForApp(application)
	return root, out, &bytes.Buffer{}
}

// newTestRootForApp builds a fresh root whose AppFactory always returns
// application, and whose stdout is the returned buffer.
func newTestRootForApp(application *app.App) (*cobra.Command, *bytes.Buffer) {
	factory := func(ctx context.Context) (*app.App, func(), error) {
		return application, func() {}, nil
	}

	out := &bytes.Buffer{}
	v, tr := false, false
	root := NewRootCommand(CommandEnv{
		Stdin:      os.Stdin,
		Stderr:     os.Stderr,
		Stdout:     out,
		AppFactory: factory,
		Verbose:    &v,
		Trace:      &tr,
	})
	return root, out
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

// TestCompanyUF_AddUpdateList asserts --uf is normalized on add, changed by
// update without touching other fields, and shown by list.
func TestCompanyUF_AddUpdateList(t *testing.T) {
	root, out, errOut := newInMemTestRoot(t)
	root.SetErr(errOut)
	certPath := filepath.Join(t.TempDir(), "company.pfx")
	if err := os.WriteFile(certPath, []byte("not inspected by company add"), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) error {
		t.Helper()
		out.Reset()
		root.SetArgs(args)
		return root.ExecuteContext(context.Background())
	}
	listedUF := func() string {
		t.Helper()
		if err := run("company", "list"); err != nil {
			t.Fatalf("company list: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		fields := strings.Fields(lines[len(lines)-1])
		return fields[len(fields)-1]
	}

	if err := run("company", "add", "--cnpj", "11222333000181", "--name", "Acme", "--cert", certPath, "--uf", "sp"); err != nil {
		t.Fatalf("company add: %v", err)
	}
	if got := listedUF(); got != "SP" {
		t.Errorf("UF after add = %q, want SP", got)
	}

	if err := run("company", "update", "--cnpj", "11.222.333/0001-81", "--uf", "RJ"); err != nil {
		t.Fatalf("company update: %v", err)
	}
	if got := listedUF(); got != "RJ" {
		t.Errorf("UF after update = %q, want RJ", got)
	}
	// company add without --env defaults to producao.
	if !strings.Contains(out.String(), "Acme") || !strings.Contains(out.String(), "producao") ||
		strings.Contains(out.String(), "producao_restrita") {
		t.Errorf("update changed fields it was not asked to: %q", out.String())
	}

	err := run("company", "add", "--cnpj", "11222333000262", "--name", "Filial", "--cert", certPath, "--uf", "XX")
	if err == nil || !strings.Contains(err.Error(), "UF") {
		t.Errorf("company add with an invalid UF = %v, want a UF error", err)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty: %q", errOut.String())
	}
}

// TestCompany_InvalidFlagsFailBeforeAppFactory asserts that company add and
// update reject a bad --env or sync start policy without opening the database.
func TestCompany_InvalidFlagsFailBeforeAppFactory(t *testing.T) {
	const factoryErr = "AppFactory não deveria rodar"
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "add with unknown env",
			args: []string{"company", "add", "--cnpj", "11222333000181", "--name", "Acme", "--env", "homologacao"},
			want: "erro no ambiente",
		},
		{
			name: "add with unknown sync start policy",
			args: []string{"company", "add", "--cnpj", "11222333000181", "--name", "Acme", "--sync-start-policy", "sometimes"},
			want: "política de sincronização inválida",
		},
		{
			name: "add with since_date and no date",
			args: []string{"company", "add", "--cnpj", "11222333000181", "--name", "Acme", "--sync-start-policy", "since_date"},
			want: "sync_start_date é obrigatório",
		},
		{
			name: "update with unknown env",
			args: []string{"company", "update", "--cnpj", "11222333000181", "--env", "homologacao"},
			want: "erro no ambiente",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// flag values stick to a command tree
			v, tr := false, false
			root := NewRootCommand(CommandEnv{
				Stdin:  os.Stdin,
				Stderr: os.Stderr,
				Stdout: &bytes.Buffer{},
				AppFactory: func(context.Context) (*app.App, func(), error) {
					return nil, nil, errors.New(factoryErr)
				},
				Verbose: &v,
				Trace:   &tr,
			})
			root.SetArgs(tc.args)
			err := root.ExecuteContext(context.Background())
			if err == nil {
				t.Fatal("Execute returned nil, want a flag error")
			}
			if strings.Contains(err.Error(), factoryErr) {
				t.Fatalf("error = %q, flags must be validated before AppFactory", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
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

func TestCompanyAndCredentialList_JSON(t *testing.T) {
	root, out, _ := newInMemTestRoot(t)
	root.SetArgs([]string{"company", "list", "--json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("company list --json: %v", err)
	}
	if got := out.String(); got != "[]\n" {
		t.Errorf("empty company list = %q, want []", got)
	}

	env := newNFeTestRoot(t)
	if err := env.run("company", "list", "--json"); err != nil {
		t.Fatalf("company list --json: %v", err)
	}
	var companies []map[string]any
	decodeJSONOutput(t, env.out, &companies)
	if len(companies) != 1 || companies[0]["CNPJ"] != nfeTestCNPJ || companies[0]["CredentialID"] != "cred-1" || companies[0]["UF"] != "SP" {
		t.Errorf("companies = %v", companies)
	}
	assertNoJSONKeys(t, companies, internalJSONKeys...)

	if err := env.run("credential", "list", "--json"); err != nil {
		t.Fatalf("credential list --json: %v", err)
	}
	var credentials []map[string]any
	decodeJSONOutput(t, env.out, &credentials)
	if len(credentials) != 1 || credentials[0]["ID"] != "cred-1" || credentials[0]["Label"] != "Mock A1" {
		t.Errorf("credentials = %v", credentials)
	}
	assertNoJSONKeys(t, credentials, internalJSONKeys...)
}

// TestCompanyAndCredentialWrites_JSON asserts that the commands that write a
// company or a credential answer --json with the record as stored, so an
// agent gets the new credential ID without parsing text.
func TestCompanyAndCredentialWrites_JSON(t *testing.T) {
	root, out, errOut := newInMemTestRoot(t)
	root.SetErr(errOut)
	certPath := filepath.Join(t.TempDir(), "a1.pfx")
	newPath := filepath.Join(t.TempDir(), "a1-novo.pfx")
	for _, p := range []string{certPath, newPath} {
		if err := os.WriteFile(p, []byte("not inspected by these commands"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(v any, args ...string) {
		t.Helper()
		out.Reset()
		root.SetArgs(append(args, "--json"))
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		decodeJSONOutput(t, out, v)
		assertNoJSONKeys(t, v, internalJSONKeys...)
	}

	var cred map[string]any
	run(&cred, "credential", "add", "--label", "Certificado A1", "--cert", certPath)
	credID, _ := cred["ID"].(string)
	if credID == "" || cred["Label"] != "Certificado A1" {
		t.Fatalf("credential add = %v, want an ID and the label", cred)
	}

	run(&cred, "credential", "update-path", "--credential-id", credID, "--cert", newPath)
	if cred["ID"] != credID {
		t.Errorf("credential update-path = %v, want ID %s", cred, credID)
	}

	var company map[string]any
	run(&company, "company", "add", "--cnpj", "11.222.333/0001-81", "--name", "Acme", "--cert", certPath, "--uf", "sp")
	if company["CNPJ"] != "11222333000181" || company["Name"] != "Acme" || company["UF"] != "SP" {
		t.Errorf("company add = %v", company)
	}

	run(&company, "company", "update", "--cnpj", "11222333000181", "--uf", "RJ")
	if company["UF"] != "RJ" || company["Name"] != "Acme" {
		t.Errorf("company update = %v, want UF RJ and the name kept", company)
	}

	run(&company, "company", "assign-credential", "--cnpj", "11222333000181", "--credential-id", credID)
	if company["CredentialID"] != credID {
		t.Errorf("company assign-credential = %v, want CredentialID %s", company, credID)
	}

	var initResult map[string]any
	run(&initResult, "init")
	if dir, _ := initResult["DataDir"].(string); dir == "" {
		t.Errorf("init = %v, want DataDir", initResult)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty: %q", errOut.String())
	}
}
