package company_test

import (
	"context"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

func TestStore_CreateCompany(t *testing.T) {
	db := storetest.OpenTestDB(t)
	s := company.NewStore(db)

	c := &nfse.Company{
		ID:              nfse.CompanyID("comp_12345"),
		CNPJ:            "11222333000181",
		Name:            "Test Company",
		Environment:     nfse.EnvironmentRestricted,
		SyncStartPolicy: nfse.SyncStartPolicyFromNow,
	}

	err := s.CreateCompany(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt to be set")
	}
}

func TestCompanyStore(t *testing.T) {
	db := storetest.OpenTestDB(t)
	repo := company.NewStore(db)
	credRepo := credential.NewStore(db)
	ctx := context.Background()

	cred := storetest.TestCredential("cred-1")
	if err := credRepo.CreateCredential(ctx, cred); err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	comp := storetest.TestCompany("comp-1", "11222333000181", nfse.EnvironmentRestricted, cred)

	// Create
	err := repo.CreateCompany(ctx, comp)
	if err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}
	if comp.CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be set")
	}

	// Fetch by CNPJ
	fetched, err := repo.CompanyByCNPJ(ctx, comp.CNPJ)
	if err != nil {
		t.Fatalf("CompanyByCNPJ failed: %v", err)
	}
	if fetched.Name != comp.Name {
		t.Errorf("Expected name %s, got %s", comp.Name, fetched.Name)
	}

	if fetched.SyncStartPolicy != nfse.SyncStartPolicyFromNow {
		t.Errorf("Expected SyncStartPolicyFromNow, got %s", fetched.SyncStartPolicy)
	}
	if fetched.SyncStartDate == nil {
		t.Errorf("Expected SyncStartDate to be set")
	}

	// Not Found
	_, err = repo.CompanyByCNPJ(ctx, "00000000000000")
	if err != company.ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound, got %v", err)
	}

	// List
	list, err := repo.ListCompanies(ctx)
	if err != nil {
		t.Fatalf("ListCompanies failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 company, got %d", len(list))
	}

	// Update
	fetched.Name = "Updated Name"
	today, err := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
	if err != nil {
		t.Fatalf("time.Parse failed: %v", err)
	}
	fetched.SyncStartDate = &today
	err = repo.UpdateCompany(ctx, fetched)
	if err != nil {
		t.Fatalf("UpdateCompany failed: %v", err)
	}

	fetched2, err := repo.CompanyByCNPJ(ctx, comp.CNPJ)
	if err != nil {
		t.Fatalf("CompanyByCNPJ failed: %v", err)
	}
	if fetched2.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", fetched2.Name)
	}

	// Assign Credential
	cred2 := storetest.TestCredential("cred-2")
	if err := credRepo.CreateCredential(ctx, cred2); err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	err = repo.AssignCredential(ctx, comp.ID, cred2.ID)
	if err != nil {
		t.Fatalf("AssignCredential failed: %v", err)
	}

	fetched3, err := repo.CompanyByCNPJ(ctx, comp.CNPJ)
	if err != nil {
		t.Fatalf("CompanyByCNPJ failed: %v", err)
	}
	if fetched3.CredentialID != cred2.ID {
		t.Errorf("Expected credential ID %s, got %s", cred2.ID, fetched3.CredentialID)
	}

	// Assign invalid credential to simulate ErrCompanyNotFound for AssignCredential
	err = repo.AssignCredential(ctx, "non-existent-company", cred2.ID)
	if err != company.ErrCompanyNotFound {
		t.Errorf("Expected ErrCompanyNotFound for non-existent company assignment, got %v", err)
	}
}
