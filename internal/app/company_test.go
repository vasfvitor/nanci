package app_test

import (
	"archive/zip"
	"context"
	"errors"
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

type credentialProviderStub struct{}

func (credentialProviderStub) GetCertPassword(context.Context, app.CertPasswordRequest) (string, error) {
	return "secret", nil
}

func newTestApp(t *testing.T) (*app.App, *store.CompanyRepository, *store.CredentialRepository, string) {
	t.Helper()

	dataDir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}

	companyRepo := store.NewCompanyRepository(db)
	credentialRepo := store.NewCredentialRepository(db)
	application, err := app.New(app.Dependencies{
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:                 db,
		CompanyRepo:        companyRepo,
		CredentialRepo:     credentialRepo,
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     store.NewDocumentRepository(db),
		XMLStore:           files.NewBlobStore(dataDir),
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		application.Close()
	})

	return application, companyRepo, credentialRepo, dataDir
}

func TestAddCompanyInheritsExistingCredentialMetadata(t *testing.T) {
	t.Parallel()

	application, companyRepo, credentialRepo, _ := newTestApp(t)
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

	err := application.AddCompany(context.Background(), app.AddCompanyInput{
		CNPJ:         "11222333000181",
		Name:         "Company",
		CredentialID: string(credential.ID),
		Environment:  string(nfse.EnvironmentProduction),
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

func TestNewRuntimeBuildsProductionDependencies(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	application, err := app.NewRuntime(app.RuntimeOptions{
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		CredentialProvider: credentialProviderStub{},
		DataDir:            dataDir,
		RunMigrations:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		application.Close()
	})

	if application.DataDir != dataDir {
		t.Fatalf("DataDir = %q, want %q", application.DataDir, dataDir)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "nanci-v1.db")); err != nil {
		t.Fatalf("expected runtime database to exist: %v", err)
	}
	if err := application.AddCredential(context.Background(), app.AddCredentialInput{
		Label:    "Credential",
		CertPath: writeTempCertFile(t),
	}); err != nil {
		t.Fatalf("expected runtime app to persist credential: %v", err)
	}
}

func TestStatusReturnsCompanyNotFound(t *testing.T) {
	t.Parallel()

	application, _, _, _ := newTestApp(t)

	_, err := application.Status(context.Background(), "11222333000181")
	if err == nil {
		t.Fatal("expected company not found error")
	}
	if !strings.Contains(err.Error(), "empresa não encontrada") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetSyncStateClearsCursorWithoutDeletingDocuments(t *testing.T) {
	t.Parallel()

	application, companyRepo, credentialRepo, _ := newTestApp(t)
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
	if _, err := application.SyncRepo.GetOrCreateState(context.Background(), nfse.GetOrCreateSyncStateParams{
		CompanyID:        company.ID,
		Environment:      company.Environment,
		ConsultationCNPJ: company.CNPJ,
		LegacyLastNSU:    42,
	}); err != nil {
		t.Fatal(err)
	}
	if err := application.SyncRepo.PersistProgress(context.Background(), nfse.PersistSyncProgressParams{
		CompanyID:             company.ID,
		RunID:                 "run-1",
		Environment:           company.Environment,
		ConsultationCNPJ:      company.CNPJ,
		LastProcessedNSU:      42,
		LastFoundNSU:          29,
		LastFoundNSUValid:     true,
		LastEmptyStreak:       3,
		CheckedCount:          1,
		DocumentsFound:        1,
		EmptyCount:            0,
		ConsecutiveEmptyCount: 0,
		ErrorsCount:           0,
		MarkSuccess:           true,
	}); err == nil {
		// PersistProgress requires an existing run; ignore and seed directly below.
	}
	if _, err := application.DB.ExecContext(context.Background(), `
		UPDATE companies SET last_nsu = 42 WHERE id = ?;
		UPDATE sync_state SET last_checked_nsu = 42, last_found_nsu = 29, last_empty_streak = 3 WHERE company_id = ?;
	`, string(company.ID), string(company.ID)); err != nil {
		t.Fatal(err)
	}

	if err := application.ResetSyncState(context.Background(), app.ResetSyncInput{CNPJ: company.CNPJ}); err != nil {
		t.Fatal(err)
	}

	stored, err := companyRepo.CompanyByCNPJ(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastNSU != 0 {
		t.Fatalf("LastNSU = %d, want 0", stored.LastNSU)
	}
	snapshot, err := application.SyncRepo.LatestSyncSnapshot(context.Background(), company.ID, company.Environment, company.CNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != nil {
		t.Fatal("expected sync_state to be removed")
	}
}

func TestUpdateCredentialPathReturnsCredentialNotFound(t *testing.T) {
	t.Parallel()

	application, _, _, _ := newTestApp(t)

	err := application.UpdateCredentialPath(context.Background(), app.UpdateCredentialPathInput{
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

	dataDir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	xmlStore := &stubXMLStore{
		data: map[string][]byte{
			"hash-1": []byte("<NFSe>stub</NFSe>"),
		},
	}
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:             db,
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
				CompanyRole: "prestada",
			},
		}},
		XMLStore:           xmlStore,
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		application.Close()
	})

	outPath := filepath.Join(t.TempDir(), "docs.zip")
	if err := application.ExportZIP(context.Background(), app.ExportInput{
		CNPJ:    "11222333000181",
		OutPath: outPath,
	}); err != nil {
		t.Fatal(err)
	}

	if len(xmlStore.getCalls) != 1 || xmlStore.getCalls[0] != "hash-1" {
		t.Fatalf("expected injected XMLStore to be used, got calls %v", xmlStore.getCalls)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	if len(reader.File) != 1 {
		t.Fatalf("expected 1 zip entry, got %d", len(reader.File))
	}
	rc, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "<NFSe>stub</NFSe>" {
		t.Fatalf("zip content = %q", string(content))
	}
}

func TestExportDANFSeUsesStoredXMLAndInjectedRenderer(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	xmlStore := &stubXMLStore{
		data: map[string][]byte{
			"hash-1": []byte("<NFSe>stub</NFSe>"),
		},
	}
	renderer := &stubDANFSeRenderer{pdf: []byte("%PDF-1.7 stub")}
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:             db,
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docByChave: &nfse.CompanyDocument{
			Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
			CompanyID:   "company-1",
			CompanyRole: "prestada",
		}},
		XMLStore:           xmlStore,
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
		DANFSeRenderer:     renderer,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		application.Close()
	})

	outPath := filepath.Join(t.TempDir(), "danfse.pdf")
	if err := application.ExportDANFSe(context.Background(), app.ExportDANFSeInput{
		CNPJ:        "11222333000181",
		ChaveAcesso: "chave-1",
		OutPath:     outPath,
	}); err != nil {
		t.Fatal(err)
	}

	if got := string(renderer.inputs[0]); got != "<NFSe>stub</NFSe>" {
		t.Fatalf("renderer input = %q", got)
	}
	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != "%PDF-1.7 stub" {
		t.Fatalf("PDF content = %q", string(written))
	}
}

func TestExportDANFSeZIPFailsWhenXMLIsMissing(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:             db,
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "missing"},
				CompanyRole: "prestada",
			},
		}},
		XMLStore:           &stubXMLStore{},
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
		DANFSeRenderer:     &stubDANFSeRenderer{pdf: []byte("%PDF-1.7 stub")},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		application.Close()
	})

	outPath := filepath.Join(t.TempDir(), "danfses.zip")
	err = application.ExportDANFSeZIP(context.Background(), app.ExportInput{
		CNPJ:    "11222333000181",
		OutPath: outPath,
	})
	if err == nil {
		t.Fatal("expected missing XML error")
	}
	if !strings.Contains(err.Error(), "ler XML original") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRejectsMissingDocumentReader(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = app.New(app.Dependencies{
		Log:                slog.Default(),
		DB:                 db,
		CompanyRepo:        store.NewCompanyRepository(db),
		CredentialRepo:     store.NewCredentialRepository(db),
		SyncRepo:           store.NewSyncRepository(db),
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

type stubXMLStore struct {
	data     map[string][]byte
	getCalls []string
}

func (s *stubXMLStore) Store(hash string, data []byte) error {
	if s.data == nil {
		s.data = make(map[string][]byte)
	}
	s.data[hash] = data
	return nil
}

func (s *stubXMLStore) Get(hash string) ([]byte, error) {
	s.getCalls = append(s.getCalls, hash)
	data, ok := s.data[hash]
	if !ok {
		return nil, errors.New("not found")
	}
	return data, nil
}

type stubCompanyRepo struct {
	company *nfse.Company
}

func (s *stubCompanyRepo) CreateCompany(context.Context, *nfse.Company) error { return nil }
func (s *stubCompanyRepo) CompanyByCNPJ(context.Context, string) (*nfse.Company, error) {
	return s.company, nil
}
func (s *stubCompanyRepo) ListCompanies(context.Context) ([]nfse.Company, error) { return nil, nil }
func (s *stubCompanyRepo) AssignCredential(context.Context, nfse.CompanyID, nfse.CredentialID) error {
	return nil
}
func (s *stubCompanyRepo) UpdateCompany(context.Context, nfse.CompanyID, string, nfse.Environment) error {
	return nil
}

type stubCredentialRepo struct{}

func (stubCredentialRepo) CreateCredential(context.Context, *nfse.Credential) error { return nil }
func (stubCredentialRepo) CredentialByID(context.Context, nfse.CredentialID) (*nfse.Credential, error) {
	return nil, store.ErrNotFound
}
func (stubCredentialRepo) ListCredentials(context.Context) ([]nfse.Credential, error) {
	return nil, nil
}
func (stubCredentialRepo) DeleteCredential(context.Context, nfse.CredentialID) error { return nil }
func (stubCredentialRepo) UpdateCredential(context.Context, *nfse.Credential) error  { return nil }

type stubDocumentReader struct {
	docs       []nfse.CompanyDocument
	docByChave *nfse.CompanyDocument
}

func (s *stubDocumentReader) ListCompanyDocuments(context.Context, nfse.CompanyID, nfse.DocumentFilter) ([]nfse.CompanyDocument, error) {
	return s.docs, nil
}

func (s *stubDocumentReader) CompanyDocumentByChave(context.Context, nfse.CompanyID, string) (*nfse.CompanyDocument, error) {
	if s.docByChave == nil {
		return nil, store.ErrNotFound
	}
	return s.docByChave, nil
}

func (s *stubDocumentReader) ListEventsByDocument(context.Context, string) ([]nfse.Event, error) {
	return nil, nil
}

type stubDANFSeRenderer struct {
	pdf    []byte
	inputs [][]byte
}

func (s *stubDANFSeRenderer) Render(xmlData []byte) ([]byte, error) {
	s.inputs = append(s.inputs, append([]byte(nil), xmlData...))
	return s.pdf, nil
}
