package app_test

import (
	"context"
	"log/slog"
	"path/filepath"
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

func TestAddCompanyInheritsExistingCredentialMetadata(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	db, err := store.OpenDB(filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}

	companyRepo := store.NewCompanyRepository(db)
	credentialRepo := store.NewCredentialRepository(db)
	credential := &nfse.Credential{
		ID:            "credential-1",
		Label:         "Certificate",
		CertPath:      `C:\certs\company.pfx`,
		Environment:   nfse.EnvironmentRestricted,
		OwnerCNPJ:     "11222333000181",
		OwnerCNPJRoot: "11222333",
	}
	if err := credentialRepo.CreateCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}

	application, err := app.New(app.Dependencies{
		Log:                slog.Default(),
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

	err = application.AddCompany(context.Background(), app.AddCompanyInput{
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
	if company.Environment != credential.Environment {
		t.Fatalf("environment = %q, want credential environment %q", company.Environment, credential.Environment)
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
