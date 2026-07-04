package credential

import (
	"context"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

func TestStore(t *testing.T) {
	db := storetest.OpenTestDB(t)
	repo := NewStore(db)
	ctx := context.Background()

	now := time.Now().UTC()
	cred := &nfse.Credential{
		ID:                "cred-1",
		Label:             "My Credential",
		CertPath:          "/path/to/cert",
		OwnerCNPJ:         "11222333000181",
		OwnerCNPJRoot:     "11222333",
		FingerprintSHA256: "hash123",
		SubjectName:       "Subject",
		NotBefore:         &now,
		NotAfter:          &now,
		InspectedAt:       &now,
	}

	err := repo.CreateCredential(ctx, cred)
	if err != nil {
		t.Fatalf("CreateCredential failed: %v", err)
	}
	if cred.CreatedAt.IsZero() {
		t.Errorf("Expected CreatedAt to be set")
	}
	if cred.UpdatedAt.IsZero() {
		t.Errorf("Expected UpdatedAt to be set")
	}

	// Fetch by ID
	fetched, err := repo.CredentialByID(ctx, cred.ID)
	if err != nil {
		t.Fatalf("CredentialByID failed: %v", err)
	}
	if fetched.Label != cred.Label {
		t.Errorf("Expected label %s, got %s", cred.Label, fetched.Label)
	}

	// Not Found
	_, err = repo.CredentialByID(ctx, "non-existent")
	if err != ErrCredentialNotFound {
		t.Errorf("Expected ErrCredentialNotFound, got %v", err)
	}

	// List
	list, err := repo.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 credential, got %d", len(list))
	}

	// Update
	cred.Label = "Updated Label"
	err = repo.UpdateCredential(ctx, cred)
	if err != nil {
		t.Fatalf("UpdateCredential failed: %v", err)
	}

	fetched, err = repo.CredentialByID(ctx, cred.ID)
	if err != nil {
		t.Fatalf("CredentialByID failed: %v", err)
	}
	if fetched.Label != "Updated Label" {
		t.Errorf("Expected label 'Updated Label', got %s", fetched.Label)
	}

	// Delete
	err = repo.DeleteCredential(ctx, cred.ID)
	if err != nil {
		t.Fatalf("DeleteCredential failed: %v", err)
	}

	_, err = repo.CredentialByID(ctx, cred.ID)
	if err != ErrCredentialNotFound {
		t.Errorf("Expected ErrCredentialNotFound, got %v", err)
	}
}
