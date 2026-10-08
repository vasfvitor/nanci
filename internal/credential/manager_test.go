package credential

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
)

type mockStore struct {
	createErr    error
	listCreds    []Credential
	listErr      error
	credByID     *Credential
	credByIDErr  error
	updateErr    error
	createdCreds []*Credential
}

func (m *mockStore) CreateCredential(ctx context.Context, c *Credential) error {
	m.createdCreds = append(m.createdCreds, c)
	return m.createErr
}

func (m *mockStore) ListCredentials(ctx context.Context) ([]Credential, error) {
	return m.listCreds, m.listErr
}

func (m *mockStore) CredentialByID(ctx context.Context, id dfe.CredentialID) (*Credential, error) {
	if m.credByIDErr != nil {
		return nil, m.credByIDErr
	}
	if m.credByID == nil {
		return nil, ErrCredentialNotFound
	}
	return m.credByID, nil
}

func (m *mockStore) UpdateCredential(ctx context.Context, c *Credential) error {
	return m.updateErr
}

func TestManager_AddCredential(t *testing.T) {
	// Create a temporary file to act as the certificate path
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.pfx")
	err := os.WriteFile(certPath, []byte("dummy cert data"), 0o600)
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
	err := os.WriteFile(certPath, []byte("dummy cert data"), 0o600)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cred := &Credential{
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
	cred := &Credential{
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
		listCreds: []Credential{{ID: "1"}, {ID: "2"}},
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

func TestManager_UpdateCredentialLookupErrors(t *testing.T) {
	certPath := filepath.Join(t.TempDir(), "test.pfx")
	if err := os.WriteFile(certPath, []byte("dummy cert data"), 0o600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	updates := []struct {
		name string
		run  func(m *Manager) error
	}{
		{"UpdateCredentialPath", func(m *Manager) error {
			return m.UpdateCredentialPath(context.Background(), UpdateCredentialPathInput{CredentialID: "cred-123", CertPath: certPath})
		}},
		{"UpdateCredentialData", func(m *Manager) error {
			return m.UpdateCredentialData(context.Background(), UpdateCredentialDataInput{CredentialID: "cred-123", Label: "New Label"})
		}},
	}
	cases := []struct {
		name  string
		store *mockStore
		check func(t *testing.T, err error)
	}{
		{
			name:  "database error",
			store: &mockStore{credByIDErr: errors.New("disk I/O error")},
			check: func(t *testing.T, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), "buscar credencial") {
					t.Errorf("error = %q, want it to mention %q", err, "buscar credencial")
				}
				if strings.Contains(err.Error(), "não encontrada") {
					t.Errorf("error = %q, a database error must not read as not found", err)
				}
			},
		},
		{
			name:  "missing credential",
			store: &mockStore{},
			check: func(t *testing.T, err error) {
				t.Helper()
				if err.Error() != "credencial não encontrada" {
					t.Errorf("error = %q, want %q", err, "credencial não encontrada")
				}
			},
		},
	}
	for _, u := range updates {
		for _, tc := range cases {
			t.Run(u.name+"/"+tc.name, func(t *testing.T) {
				err := u.run(NewManager(tc.store))
				if err == nil {
					t.Fatal("expected error")
				}
				tc.check(t, err)
			})
		}
	}
}
