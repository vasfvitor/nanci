package app_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

func setupTestApp(t *testing.T) *app.App {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, "file::memory:?cache=shared", true)
	if err != nil {
		t.Fatalf("falha ao abrir db em memoria: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	application := &app.App{
		CompanyRepo:    store.NewCompanyRepository(db),
		CredentialRepo: store.NewCredentialRepository(db),
		SyncRepo:       store.NewSyncRepository(db),
	}
	return application
}

func TestAppIntegration_OnboardingFlow(t *testing.T) {
	application := setupTestApp(t)
	ctx := context.Background()

	// Use this test file as the certificate path to pass os.Stat checks
	certPath, _ := filepath.Abs("app_integration_test.go")

	// 1. Add Credential
	credInput := app.AddCredentialInput{
		Label:    "Test Cert",
		CertPath: certPath,
	}

	err := application.AddCredential(ctx, credInput)
	if err != nil {
		t.Fatalf("AddCredential falhou: %v", err)
	}

	creds, err := application.ListCredentials(ctx)
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
	err = application.AddCompany(ctx, compInput)
	if err != nil {
		t.Fatalf("AddCompany falhou: %v", err)
	}

	// 3. Validate
	comps, err := application.ListCompanies(ctx)
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
