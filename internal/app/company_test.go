package app_test

import (
	"archive/zip"
	"context"
	"database/sql"
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

func int64Ptr(v int64) *int64 { return &v }

type credentialProviderStub struct{}

func (credentialProviderStub) GetCertPassword(context.Context, app.CertPasswordRequest) (string, error) {
	return "secret", nil
}

func newTestApp(t *testing.T) (*app.App, app.CompanyRepository, app.CredentialRepository, string, *sql.DB) {
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
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:        companyRepo,
		CredentialRepo:     credentialRepo,
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     store.NewDocumentRepository(db),
		DocumentTracker:    store.NewDocumentRepository(db),
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

	err := application.AddCompany(context.Background(), app.AddCompanyInput{
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
	if _, err := application.SyncRepo.GetOrCreateState(context.Background(), nfse.GetOrCreateSyncStateParams{
		CompanyID:        company.ID,
		Environment:      company.Environment,
		ConsultationCNPJ: company.CNPJ,
	}); err != nil {
		t.Fatal(err)
	}
	_ = application.SyncRepo.PersistProgress(context.Background(), nfse.PersistSyncProgressParams{
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

	if err := application.ResetSyncState(context.Background(), app.ResetSyncInput{CNPJ: company.CNPJ}); err != nil {
		t.Fatal(err)
	}

	_, err := companyRepo.CompanyByCNPJ(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatal(err)
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

	application, _, _, _, _ := newTestApp(t)

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
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(func() { _ = db.Close() })
	xmlStore := &stubXMLStore{
		data: map[string][]byte{
			"hash-1": []byte("<NFSe>stub</NFSe>"),
		},
	}
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
				CompanyRole: "prestada",
			},
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           xmlStore,
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "docs.zip")
	if _, err := application.ExportZIP(context.Background(), app.ExportInput{
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
	defer func() { _ = reader.Close() }()

	if len(reader.File) != 1 {
		t.Fatalf("expected 1 zip entry, got %d", len(reader.File))
	}
	rc, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()

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
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	xmlStore := &stubXMLStore{
		data: map[string][]byte{
			"hash-1": []byte("<NFSe>stub</NFSe>"),
		},
	}
	renderer := &stubDANFSeRenderer{pdf: []byte("%PDF-1.7 stub")}
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docByChave: &nfse.CompanyDocument{
			Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
			CompanyID:   "company-1",
			CompanyRole: "prestada",
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           xmlStore,
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
		DANFSeRenderer:     renderer,
	})
	if err != nil {
		t.Fatal(err)
	}

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
	written, err := os.ReadFile(outPath) //nolint:gosec // intentional: test file reading
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
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "missing"},
				CompanyRole: "prestada",
			},
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           &stubXMLStore{},
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
		DANFSeRenderer:     &stubDANFSeRenderer{pdf: []byte("%PDF-1.7 stub")},
	})
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "danfses.zip")
	_, err = application.ExportDANFSeZIP(context.Background(), app.ExportInput{
		CNPJ:    "11222333000181",
		OutPath: outPath,
	})
	if err == nil {
		t.Fatal("expected missing XML error")
	}
	if !strings.Contains(err.Error(), "ler XML original") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.Is(err, files.ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestExportDANFSeZIPPreservesRendererFailures(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
				CompanyRole: "prestada",
			},
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           &stubXMLStore{data: map[string][]byte{"hash-1": []byte("<NFSe/>")}},
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
		DANFSeRenderer:     &stubDANFSeRenderer{err: errors.New("renderer exploded")},
	})
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "danfses.zip")
	_, err = application.ExportDANFSeZIP(context.Background(), app.ExportInput{
		CNPJ:    "11222333000181",
		OutPath: outPath,
	})
	if err == nil {
		t.Fatal("expected renderer error")
	}
	if !strings.Contains(err.Error(), "renderizar DANFSe") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "arquivo físico XML não encontrado") {
		t.Fatalf("renderer error should not be rewritten as missing XML: %v", err)
	}
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
		Log:                slog.Default(),
		CompanyRepo:        store.NewCompanyRepository(db),
		CredentialRepo:     store.NewCredentialRepository(db),
		SyncRepo:           store.NewSyncRepository(db),
		DocumentTracker:    store.NewDocumentRepository(db),
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
		return nil, files.ErrFileNotFound
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
func (s *stubCompanyRepo) UpdateCompany(context.Context, *nfse.Company) error {
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

func (s *stubDocumentReader) ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error) {
	return nil, nil
}

func (s *stubDocumentReader) CountDocumentsByRole(ctx context.Context, companyID nfse.CompanyID) (map[string]int64, error) {
	return map[string]int64{}, nil
}

type stubDANFSeRenderer struct {
	pdf    []byte
	inputs [][]byte
	err    error
}

func (s *stubDANFSeRenderer) Render(xmlData []byte) ([]byte, error) {
	s.inputs = append(s.inputs, append([]byte(nil), xmlData...))
	if s.err != nil {
		return nil, s.err
	}
	return s.pdf, nil
}

func TestExportCSV_Success(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
				CompanyRole: "prestada",
			},
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           &stubXMLStore{},
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "docs.csv")
	res, err := application.ExportCSV(context.Background(), app.ExportInput{
		CNPJ:    "11222333000181",
		OutPath: outPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.ExportedCount != 1 {
		t.Fatalf("expected 1 exported document, got %d", res.ExportedCount)
	}

	content, err := os.ReadFile(outPath) //nolint:gosec // intentional test read
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "chave-1") {
		t.Fatalf("expected CSV to contain 'chave-1'")
	}
}

func TestExportXLSX_Success(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docs: []nfse.CompanyDocument{
			{
				Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
				CompanyRole: "prestada",
			},
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           &stubXMLStore{},
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "docs.xlsx")
	res, err := application.ExportXLSX(context.Background(), app.ExportInput{
		CNPJ:    "11222333000181",
		OutPath: outPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.ExportedCount != 1 {
		t.Fatalf("expected 1 exported document, got %d", res.ExportedCount)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected XLSX file to exist, got error: %v", err)
	}
}

func TestExportXML_Success(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	xmlStore := &stubXMLStore{
		data: map[string][]byte{
			"hash-1": []byte("<NFSe>stub</NFSe>"),
		},
	}

	application, err := app.New(app.Dependencies{
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:    &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo: &stubCredentialRepo{},
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: &stubDocumentReader{docByChave: &nfse.CompanyDocument{
			Document:    nfse.Document{ChaveAcesso: "chave-1", Competence: "2026-06", RawHash: "hash-1"},
			CompanyRole: "prestada",
		}},
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           xmlStore,
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "doc.xml")
	err = application.ExportXML(context.Background(), app.ExportXMLInput{
		CNPJ:        "11222333000181",
		ChaveAcesso: "chave-1",
		OutPath:     outPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(outPath) //nolint:gosec // intentional test read
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "<NFSe>stub</NFSe>" {
		t.Fatalf("expected XML content to be '<NFSe>stub</NFSe>', got: %s", string(content))
	}
}

type stubDocumentTracker struct {
	count int
}

func (s *stubDocumentTracker) ListPendingExportDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter, kind string) ([]nfse.CompanyDocument, error) {
	return nil, nil
}

func (s *stubDocumentTracker) CountPendingExportDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter, kind string) (int, error) {
	return s.count, nil
}

func (s *stubDocumentTracker) MarkDocumentsExported(ctx context.Context, companyID nfse.CompanyID, kind string, marks []nfse.DocumentExportMark) error {
	return nil
}

func (s *stubDocumentTracker) MarkDocumentsViewed(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter) (int, error) {
	return 0, nil
}

func TestCountPendingExportDocuments(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	application, err := app.New(app.Dependencies{
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)), //nolint:sloglint
		CompanyRepo:        &stubCompanyRepo{company: &nfse.Company{ID: "company-1", CNPJ: "11222333000181", Name: "Company"}},
		CredentialRepo:     &stubCredentialRepo{},
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     &stubDocumentReader{},
		DocumentTracker:    &stubDocumentTracker{count: 42},
		XMLStore:           &stubXMLStore{},
		DataDir:            dataDir,
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatal(err)
	}

	count, err := application.CountPendingExportDocuments(context.Background(), app.ExportInput{
		CNPJ: "11222333000181",
	}, "csv")
	if err != nil {
		t.Fatal(err)
	}

	if count != 42 {
		t.Fatalf("expected 42 pending documents, got %d", count)
	}
}
