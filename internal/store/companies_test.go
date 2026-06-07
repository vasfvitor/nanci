package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewSQLiteStore(dbPath, true) // Cria e roda migrations
	if err != nil {
		t.Fatalf("falha ao criar test store: %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

func TestSQLiteStore_CompanyCRUD(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	credentialID := createTestCredential(t, store, "cred-1", "/tmp/cert.pfx", "producao_restrita")

	// 1. Create Company
	company := &nfse.Company{
		ID:           "comp_123",
		CNPJ:         "12345678000199",
		CNPJRoot:     "12345678",
		Name:         "Test Company",
		CredentialID: credentialID,
		LastNSU:      0,
	}

	err := store.CreateCompany(ctx, company)
	if err != nil {
		t.Fatalf("CreateCompany error: %v", err)
	}

	if company.CreatedAt.IsZero() || company.UpdatedAt.IsZero() {
		t.Error("CreatedAt or UpdatedAt not set")
	}

	// 2. Get Company
	c, err := store.GetCompany(ctx, "12345678000199")
	if err != nil {
		t.Fatalf("GetCompany error: %v", err)
	}
	if c == nil {
		t.Fatal("Company not found")
	}
	if c.Name != "Test Company" {
		t.Errorf("expected Name 'Test Company', got '%s'", c.Name)
	}
	if c.CredentialID != credentialID {
		t.Fatalf("CredentialID = %q, want %q", c.CredentialID, credentialID)
	}

	// 3. List Companies
	companies, err := store.ListCompanies(ctx)
	if err != nil {
		t.Fatalf("ListCompanies error: %v", err)
	}
	if len(companies) != 1 {
		t.Errorf("expected 1 company, got %d", len(companies))
	}

	// 4. Update LastNSU
	err = store.UpdateLastNSU(ctx, "comp_123", 150)
	if err != nil {
		t.Fatalf("UpdateLastNSU error: %v", err)
	}

	c2, _ := store.GetCompany(ctx, "12345678000199")
	if c2.LastNSU != 150 {
		t.Errorf("expected LastNSU 150, got %d", c2.LastNSU)
	}
}

func createTestCredential(t *testing.T, store *SQLiteStore, id, certPath, environment string) string {
	t.Helper()

	now := time.Now().UTC()
	credential := &nfse.Credential{
		ID:            id,
		Label:         id,
		CertPath:      certPath,
		Environment:   environment,
		OwnerCNPJ:     "12345678000199",
		OwnerCNPJRoot: "12345678",
		NotBefore:     &now,
		NotAfter:      &now,
		InspectedAt:   &now,
	}
	if err := store.CreateCredential(context.Background(), credential); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
	return credential.ID
}
