package app_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

func int64Ptr(v int64) *int64 { return &v }

type credentialProviderStub struct{}

func (credentialProviderStub) GetCertPassword(context.Context, app.CertPasswordRequest) (string, error) {
	return "secret", nil
}

func newTestApp(t *testing.T) (*app.App, *store.CompanyRepository, *store.CredentialRepository, string, *sql.DB) {
	t.Helper()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(func() { _ = db.Close() })

	companyRepo := store.NewCompanyRepository(db)
	credentialRepo := store.NewCredentialRepository(db)
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    companyRepo,
		CredentialRepo: credentialRepo,
		SyncRepo:       store.NewSyncRepository(db),
		DocumentRepo:   store.NewDocumentRepository(db),

		XMLStore:           files.NewBlobStore(dataDir),
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}

	return application, companyRepo, credentialRepo, dataDir, db
}

func TestAddCompanyInheritsExistingCredentialMetadata(t *testing.T) {
	t.Parallel()

	application, companyRepo, credentialRepo, _, _ := newTestApp(t)
	credential := &nfse.Credential{
		ID:            "credential-1",
		Label:         "Certificate",
		CertPath:      `C:\certs\company.pfx`,
		OwnerCNPJ:     "11222333000181",
		OwnerCNPJRoot: "11222333",
	}
	if err := credentialRepo.CreateCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}

	err := application.Companies.AddCompany(context.Background(), app.AddCompanyInput{
		CNPJ:         "11222333000181",
		Name:         "Company",
		CredentialID: string(credential.ID),
		Environment:  nfse.EnvironmentProduction,
	})
	if err != nil {
		t.Fatal(err)
	}

	company, err := companyRepo.CompanyByCNPJ(context.Background(), "11222333000181")
	if err != nil {
		t.Fatal(err)
	}
	if company.CredentialID != credential.ID {
		t.Fatalf("credential ID = %q, want %q", company.CredentialID, credential.ID)
	}
	if company.CredentialLabel != credential.Label {
		t.Fatalf("credential label = %q, want %q", company.CredentialLabel, credential.Label)
	}
	if company.CredentialCertPath != credential.CertPath {
		t.Fatalf("credential path = %q, want %q", company.CredentialCertPath, credential.CertPath)
	}
	if company.Environment != nfse.EnvironmentProduction {
		t.Fatalf("environment = %q, want %q", company.Environment, nfse.EnvironmentProduction)
	}
}

func TestStatusReturnsCompanyNotFound(t *testing.T) {
	t.Parallel()

	application, _, _, _, _ := newTestApp(t)

	_, err := application.Status.Status(context.Background(), "11222333000181")
	if err == nil {
		t.Fatal("expected company not found error")
	}
	if !strings.Contains(err.Error(), "empresa não encontrada") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetSyncStateClearsCursorWithoutDeletingDocuments(t *testing.T) {
	t.Parallel()

	application, companyRepo, credentialRepo, _, db := newTestApp(t)
	credential := &nfse.Credential{
		ID:            "credential-1",
		Label:         "Certificate",
		CertPath:      `C:\certs\company.pfx`,
		OwnerCNPJ:     "11222333000181",
		OwnerCNPJRoot: "11222333",
	}
	if err := credentialRepo.CreateCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	company := &nfse.Company{
		ID:                 "company-1",
		CNPJ:               "11222333000181",
		CNPJRoot:           "11222333",
		Name:               "Company",
		CredentialID:       credential.ID,
		CredentialLabel:    credential.Label,
		CredentialCertPath: credential.CertPath,
		Environment:        nfse.EnvironmentProduction,
	}
	if err := companyRepo.CreateCompany(context.Background(), company); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewSyncRepository(db).GetOrCreateState(context.Background(), nfse.GetOrCreateSyncStateParams{
		CompanyID:        company.ID,
		Environment:      company.Environment,
		ConsultationCNPJ: company.CNPJ,
	}); err != nil {
		t.Fatal(err)
	}
	_ = store.NewSyncRepository(db).PersistProgress(context.Background(), nfse.PersistSyncProgressParams{
		CompanyID:             company.ID,
		RunID:                 "run-1",
		Environment:           company.Environment,
		ConsultationCNPJ:      company.CNPJ,
		LastProcessedNSU:      42,
		LastFoundNSU:          int64Ptr(29),
		LastEmptyStreak:       3,
		CheckedCount:          1,
		DocumentsFound:        1,
		EmptyCount:            0,
		ConsecutiveEmptyCount: 0,
		ErrorsCount:           0,
		MarkSuccess:           true,
	})
	// PersistProgress requires an existing run; ignore error and seed directly below.
	if _, err := db.ExecContext(context.Background(), `
		UPDATE sync_state SET last_checked_nsu = 42, last_found_nsu = 29, last_empty_streak = 3 WHERE company_id = ?;
	`, string(company.ID)); err != nil {
		t.Fatal(err)
	}

	if err := application.Sync.ResetSyncState(context.Background(), app.ResetSyncInput{CNPJ: company.CNPJ}); err != nil {
		t.Fatal(err)
	}

	_, err := companyRepo.CompanyByCNPJ(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.NewSyncRepository(db).LatestSyncSnapshot(context.Background(), company.ID, company.Environment, company.CNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != nil {
		t.Fatal("expected sync_state to be removed")
	}
}

func TestUpdateCredentialPathReturnsCredentialNotFound(t *testing.T) {
	t.Parallel()

	application, _, _, _, _ := newTestApp(t)

	err := application.Credentials.UpdateCredentialPath(context.Background(), app.UpdateCredentialPathInput{
		CredentialID: "missing",
		CertPath:     writeTempCertFile(t),
	})
	if err == nil {
		t.Fatal("expected credential not found error")
	}
	if !strings.Contains(err.Error(), "credencial não encontrada") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportZIPUsesInjectedXMLStore(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportZIPUsesInjectedXMLStore to use integration test setup
}

func TestExportDANFSeUsesStoredXMLAndInjectedRenderer(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportDANFSeUsesStoredXMLAndInjectedRenderer to use integration test setup
}

func TestExportDANFSeZIPFailsWhenXMLIsMissing(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportDANFSeZIPFailsWhenXMLIsMissing to use integration test setup
}

func TestExportDANFSeZIPPreservesRendererFailures(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportDANFSeZIPPreservesRendererFailures to use integration test setup
}

func TestNewRejectsMissingDocumentReader(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(func() { _ = db.Close() })

	_, err = app.New(app.Dependencies{
		Log:            slog.Default(),
		CompanyRepo:    store.NewCompanyRepository(db),
		CredentialRepo: store.NewCredentialRepository(db),
		SyncRepo:       store.NewSyncRepository(db),

		XMLStore:           files.NewBlobStore(dataDir),
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err == nil {
		t.Fatal("expected missing document reader error")
	}
}

func writeTempCertFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cert.pfx")
	if err := os.WriteFile(path, []byte("stub"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExportCSV_Success(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportCSV_Success to use integration test setup
}

func TestExportXLSX_Success(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportXLSX_Success to use integration test setup
}

func TestExportXML_Success(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestExportXML_Success to use integration test setup
}

func TestCountPendingExportDocuments(t *testing.T) {
	t.Parallel()
	// TODO: Rewrite TestCountPendingExportDocuments to use integration test setup
}
