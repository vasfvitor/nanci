package credential

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

type mockStore struct {
	createErr    error
	listCreds    []nfse.Credential
	listErr      error
	credByID     *nfse.Credential
	credByIDErr  error
	updateErr    error
	createdCreds []*nfse.Credential
}

func (m *mockStore) CreateCredential(ctx context.Context, c *nfse.Credential) error {
	m.createdCreds = append(m.createdCreds, c)
	return m.createErr
}

func (m *mockStore) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	return m.listCreds, m.listErr
}

func (m *mockStore) CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error) {
	if m.credByIDErr != nil {
		return nil, m.credByIDErr
	}
	if m.credByID == nil {
		return nil, ErrCredentialNotFound
	}
	return m.credByID, nil
}

func (m *mockStore) UpdateCredential(ctx context.Context, c *nfse.Credential) error {
	return m.updateErr
}

func TestManager_AddCredential(t *testing.T) {
	// Create a temporary file to act as the certificate path
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.pfx")
	err := os.WriteFile(certPath, []byte("dummy cert data"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	mock := &mockStore{}
	m := NewManager(mock)

	input := AddCredentialInput{
		Label:    "Test Label",
		CertPath: certPath,
	}

	err = m.AddCredential(context.Background(), input)
	if err != nil {
		t.Fatalf("AddCredential failed: %v", err)
	}

	if len(mock.createdCreds) != 1 {
		t.Fatalf("Expected 1 credential to be created, got %d", len(mock.createdCreds))
	}

	cred := mock.createdCreds[0]
	if cred.Label != input.Label {
		t.Errorf("Expected label %s, got %s", input.Label, cred.Label)
	}
	if cred.CertPath != input.CertPath {
		t.Errorf("Expected cert path %s, got %s", input.CertPath, cred.CertPath)
	}
}

func TestManager_UpdateCredentialPath(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.pfx")
	err := os.WriteFile(certPath, []byte("dummy cert data"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cred := &nfse.Credential{
		ID:       "cred-123",
		Label:    "Test",
		CertPath: "/old/path",
	}
	mock := &mockStore{
		credByID: cred,
	}
	m := NewManager(mock)

	input := UpdateCredentialPathInput{
		CredentialID: string(cred.ID),
		CertPath:     certPath,
	}

	err = m.UpdateCredentialPath(context.Background(), input)
	if err != nil {
		t.Fatalf("UpdateCredentialPath failed: %v", err)
	}

	if cred.CertPath != certPath {
		t.Errorf("Expected cert path to be updated to %s, got %s", certPath, cred.CertPath)
	}
}

func TestManager_UpdateCredentialData(t *testing.T) {
	cred := &nfse.Credential{
		ID:    "cred-123",
		Label: "Old Label",
	}
	mock := &mockStore{
		credByID: cred,
	}
	m := NewManager(mock)

	input := UpdateCredentialDataInput{
		CredentialID: string(cred.ID),
		Label:        "New Label",
	}

	err := m.UpdateCredentialData(context.Background(), input)
	if err != nil {
		t.Fatalf("UpdateCredentialData failed: %v", err)
	}

	if cred.Label != "New Label" {
		t.Errorf("Expected label to be updated to 'New Label', got %s", cred.Label)
	}
}

func TestManager_ListCredentials(t *testing.T) {
	mock := &mockStore{
		listCreds: []nfse.Credential{{ID: "1"}, {ID: "2"}},
	}
	m := NewManager(mock)

	creds, err := m.ListCredentials(context.Background())
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}

	if len(creds) != 2 {
		t.Errorf("Expected 2 credentials, got %d", len(creds))
	}
}
