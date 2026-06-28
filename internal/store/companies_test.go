package store

import (
	"context"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestCompanyRepository(t *testing.T) {
	db := openTestDB(t)
	repo := NewCompanyRepository(db)
	credRepo := NewCredentialRepository(db)
	ctx := context.Background()

	cred := testCredential("cred-1")
	if err := credRepo.CreateCredential(ctx, cred); err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	company := testCompany("comp-1", "11222333000181", nfse.EnvironmentRestricted, cred)
	
	// Create
	err := repo.CreateCompany(ctx, company)
	if err != nil {
		t.Fatalf("CreateCompany failed: %v", err)
	}
	if company.CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be set")
	}

	// Fetch by CNPJ
	fetched, err := repo.CompanyByCNPJ(ctx, company.CNPJ)
	if err != nil {
		t.Fatalf("CompanyByCNPJ failed: %v", err)
	}
	if fetched.Name != company.Name {
		t.Errorf("Expected name %s, got %s", company.Name, fetched.Name)
	}
	
	if fetched.SyncStartPolicy != nfse.SyncStartPolicyFromNow {
		t.Errorf("Expected SyncStartPolicyFromNow, got %s", fetched.SyncStartPolicy)
	}
	if fetched.SyncStartDate == nil {
		t.Errorf("Expected SyncStartDate to be set")
	}

	// Not Found
	_, err = repo.CompanyByCNPJ(ctx, "00000000000000")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
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

	fetched2, err := repo.CompanyByCNPJ(ctx, company.CNPJ)
	if err != nil {
		t.Fatalf("CompanyByCNPJ failed: %v", err)
	}
	if fetched2.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", fetched2.Name)
	}

	// Assign Credential
	cred2 := testCredential("cred-2")
	if err := credRepo.CreateCredential(ctx, cred2); err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}

	err = repo.AssignCredential(ctx, company.ID, cred2.ID)
	if err != nil {
		t.Fatalf("AssignCredential failed: %v", err)
	}

	fetched3, err := repo.CompanyByCNPJ(ctx, company.CNPJ)
	if err != nil {
		t.Fatalf("CompanyByCNPJ failed: %v", err)
	}
	if fetched3.CredentialID != cred2.ID {
		t.Errorf("Expected credential ID %s, got %s", cred2.ID, fetched3.CredentialID)
	}

	// Assign invalid credential to simulate ErrNotFound for AssignCredential
	err = repo.AssignCredential(ctx, "non-existent-company", cred2.ID)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent company assignment, got %v", err)
	}
}
