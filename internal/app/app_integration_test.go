package app_test

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

func setupTestApp(t *testing.T) (*app.App, *sql.DB) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, "file::memory:?cache=shared", true)
	if err != nil {
		t.Fatalf("falha ao abrir db em memoria: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	docRepo := store.NewDocumentRepository(db)
	application, err := app.New(app.Dependencies{
		Log:                slog.Default(),
		CompanyRepo:        store.NewCompanyRepository(db),
		CredentialRepo:     store.NewCredentialRepository(db),
		SyncRepo:           store.NewSyncRepository(db),
		DocumentRepo:       docRepo,
		XMLStore:           files.NewBlobStore(t.TempDir()),
		DataDir:            t.TempDir(),
		CredentialProvider: credentialProviderStub{},
	})
	if err != nil {
		t.Fatalf("falha ao criar app: %v", err)
	}
	return application, db
}

func TestAppIntegration_OnboardingFlow(t *testing.T) {
	application, _ := setupTestApp(t)
	ctx := context.Background()

	// Use this test file as the certificate path to pass os.Stat checks
	certPath, _ := filepath.Abs("app_integration_test.go")

	// 1. Add Credential
	credInput := app.AddCredentialInput{
		Label:    "Test Cert",
		CertPath: certPath,
	}

	err := application.Credentials.AddCredential(ctx, credInput)
	if err != nil {
		t.Fatalf("AddCredential falhou: %v", err)
	}

	creds, err := application.Credentials.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("ListCredentials falhou: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("esperava 1 credencial, obteve %d", len(creds))
	}
	credID := creds[0].ID

	// 2. Add Company
	compInput := app.AddCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Empresa Teste",
		Environment:     nfse.EnvironmentRestricted,
		CredentialID:    string(credID),
		SyncStartPolicy: nfse.SyncStartPolicyAll,
	}
	err = application.Companies.AddCompany(ctx, compInput)
	if err != nil {
		t.Fatalf("AddCompany falhou: %v", err)
	}

	// 3. Validate
	comps, err := application.Companies.ListCompanies(ctx)
	if err != nil {
		t.Fatalf("ListCompanies falhou: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("esperava 1 empresa, obteve %d", len(comps))
	}

	if comps[0].CredentialID != credID {
		t.Errorf("esperava CredentialID %s, obteve %s", credID, comps[0].CredentialID)
	}
}

func TestAppIntegration_SyncPreferencesFlow(t *testing.T) {
	application, _ := setupTestApp(t)
	ctx := context.Background()

	// Use this test file as the certificate path to pass os.Stat checks
	certPath, _ := filepath.Abs("app_integration_test.go")

	// Setup base company
	application.Credentials.AddCredential(ctx, app.AddCredentialInput{Label: "L", CertPath: certPath})
	creds, _ := application.Credentials.ListCredentials(ctx)
	application.Companies.AddCompany(ctx, app.AddCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Empresa Sync",
		Environment:     nfse.EnvironmentRestricted,
		CredentialID:    string(creds[0].ID),
		SyncStartPolicy: "all",
	})

	// Act: Update to from_now
	policyFromNow, dateFromNow, _ := app.ParseSyncStartPolicyInput("from_now", "")
	updateInput := app.UpdateCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Empresa Sync",
		Environment:     nfse.EnvironmentRestricted,
		SyncStartPolicy: policyFromNow,
		SyncStartDate:   dateFromNow,
	}
	err := application.Companies.UpdateCompany(ctx, updateInput)
	if err != nil {
		t.Fatalf("UpdateCompany falhou: %v", err)
	}

	// Assert
	comps, _ := application.Companies.ListCompanies(ctx)
	if comps[0].SyncStartPolicy != nfse.SyncStartPolicyFromNow {
		t.Errorf("esperava politica from_now, obteve %s", comps[0].SyncStartPolicy)
	}
	if comps[0].SyncStartDate == nil {
		t.Error("esperava SyncStartDate preenchido para from_now, mas veio nil")
	} else {
		expectedDate := time.Now().Truncate(24 * time.Hour)
		actualDate := comps[0].SyncStartDate.Truncate(24 * time.Hour)
		if !actualDate.Equal(expectedDate) {
			t.Errorf("esperava SyncStartDate %v, obteve %v", expectedDate, actualDate)
		}
	}

	// Act: Update back to all
	policyAll, dateAll, _ := app.ParseSyncStartPolicyInput("all", "")
	updateInput.SyncStartPolicy = policyAll
	updateInput.SyncStartDate = dateAll
	application.Companies.UpdateCompany(ctx, updateInput)
	comps, _ = application.Companies.ListCompanies(ctx)

	if comps[0].SyncStartPolicy != nfse.SyncStartPolicyAll {
		t.Errorf("esperava politica all, obteve %s", comps[0].SyncStartPolicy)
	}
	if comps[0].SyncStartDate != nil {
		t.Errorf("esperava SyncStartDate nil para all, obteve %v", comps[0].SyncStartDate)
	}
}
