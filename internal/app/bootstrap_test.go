package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
)

func completeDependencies(t *testing.T) Dependencies {
	t.Helper()
	db := storetest.OpenTestDB(t)
	return Dependencies{
		Log:                slog.New(slog.DiscardHandler),
		CompanyStore:       company.NewStore(db),
		CredentialStore:    credential.NewStore(db),
		SyncRepo:           sync.NewStore(db),
		DocumentRepo:       store.NewDocumentRepository(db),
		NFeRepo:            store.NewNFeRepository(db),
		CTeRepo:            store.NewCTeRepository(db),
		XMLStore:           files.NewBlobStore(t.TempDir()),
		DataDir:            t.TempDir(),
		CredentialProvider: &nfePasswordStub{},
	}
}

func TestNewRejectsMissingDependencies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		clear   func(*Dependencies)
		wantErr string
	}{
		{"logger", func(d *Dependencies) { d.Log = nil }, "app: logger is required"},
		{"companies", func(d *Dependencies) { d.CompanyStore = nil }, "app: company repository is required"},
		{"credentials", func(d *Dependencies) { d.CredentialStore = nil }, "app: credential repository is required"},
		{"sync", func(d *Dependencies) { d.SyncRepo = nil }, "app: sync repository is required"},
		{"documents", func(d *Dependencies) { d.DocumentRepo = nil }, "app: document repository is required"},
		{"nfe", func(d *Dependencies) { d.NFeRepo = nil }, "app: NF-e repository is required"},
		{"cte", func(d *Dependencies) { d.CTeRepo = nil }, "app: CT-e repository is required"},
		{"xml", func(d *Dependencies) { d.XMLStore = nil }, "app: XML store is required"},
		{"data dir", func(d *Dependencies) { d.DataDir = "" }, "app: data directory is required"},
		{"passwords", func(d *Dependencies) { d.CredentialProvider = nil }, "app: credential provider is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := completeDependencies(t)
			tc.clear(&deps)

			application, err := New(deps)
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("New error = %v, want %q", err, tc.wantErr)
			}
			if application != nil {
				t.Errorf("New returned an App with an error: %+v", application)
			}
		})
	}
}

// TestNewRuntimeWiresServices: the services share the dependencies they were
// given, and the optional renderers may be nil.
func TestNewRuntimeWiresServices(t *testing.T) {
	t.Parallel()
	deps := completeDependencies(t)

	application, err := NewRuntime(deps)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	if application.Companies == nil || application.Credentials == nil || application.Documents == nil ||
		application.Exports == nil || application.Query == nil || application.NFe == nil || application.CTe == nil {
		t.Fatalf("NewRuntime left a service nil: %+v", application)
	}
	if application.SyncManager == nil || application.SyncManager.CompanyProvider != deps.CompanyStore || application.SyncManager.XMLStore != deps.XMLStore {
		t.Errorf("sync manager not wired to the given stores: %+v", application.SyncManager)
	}
	if application.Query.CredentialStore != deps.CredentialStore || application.Query.Certificates.Passwords != deps.CredentialProvider {
		t.Errorf("query service not wired to the given credentials: %+v", application.Query)
	}
	if application.Exports.DocumentRepo != deps.DocumentRepo || application.Exports.DANFSeRenderer != nil {
		t.Errorf("export service = %+v, want the given repository and no renderer", application.Exports)
	}

	if _, err := NewRuntime(Dependencies{}); err == nil {
		t.Error("NewRuntime accepted empty dependencies")
	}
}

func TestResolveRuntimeDataDir(t *testing.T) {
	envDir := t.TempDir()
	t.Setenv("NANCI_DATA_DIR", envDir)

	got, err := ResolveRuntimeDataDir("")
	if err != nil || got != envDir {
		t.Errorf("ResolveRuntimeDataDir(\"\") = %q, %v; want NANCI_DATA_DIR %q", got, err, envDir)
	}

	override := filepath.Join(t.TempDir(), "outro")
	got, err = ResolveRuntimeDataDir(override)
	if err != nil || got != override {
		t.Errorf("ResolveRuntimeDataDir(override) = %q, %v; want %q", got, err, override)
	}
}

func TestRuntimeDBPath(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("dados", "nanci")
	if got, want := RuntimeDBPath(dir), filepath.Join(dir, "nanci-v1.db"); got != want {
		t.Errorf("RuntimeDBPath = %q, want %q", got, want)
	}
}

// TestLoadRuntimeEnv reads .env.local from the data directory and keeps the
// variables the process already has.
func TestLoadRuntimeEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	dataDir := t.TempDir()
	t.Setenv("NANCI_DATA_DIR", dataDir)
	const fromFile, kept = "NANCI_TEST_RUNTIME_ENV_FILE", "NANCI_TEST_RUNTIME_ENV_KEPT"
	t.Setenv(fromFile, "")
	if err := os.Unsetenv(fromFile); err != nil { // t.Setenv restores the unset state afterwards
		t.Fatal(err)
	}
	t.Setenv(kept, "do-processo")
	content := "# local\n" + fromFile + "=do-arquivo\n" + kept + "=do-arquivo\n"
	if err := os.WriteFile(filepath.Join(dataDir, ".env.local"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := LoadRuntimeEnv(); err != nil {
		t.Fatalf("LoadRuntimeEnv: %v", err)
	}
	if got := os.Getenv(fromFile); got != "do-arquivo" {
		t.Errorf("%s = %q, want the value from the file", fromFile, got)
	}
	if got := os.Getenv(kept); got != "do-processo" {
		t.Errorf("%s = %q, want the process value to win", kept, got)
	}
}

func TestLoadRuntimeEnvWrapsParseErrors(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	t.Setenv("NANCI_DATA_DIR", t.TempDir())
	if err := os.WriteFile(filepath.Join(cwd, ".env.local"), []byte("SEM_IGUAL\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := LoadRuntimeEnv()
	if err == nil || !strings.HasPrefix(err.Error(), "carregar .env.local: ") || !strings.Contains(err.Error(), "missing '='") {
		t.Fatalf("LoadRuntimeEnv error = %v, want the wrapped parse error", err)
	}
}
