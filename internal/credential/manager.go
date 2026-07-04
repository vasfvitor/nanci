package credential

import (
	"context"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// AddCredentialInput carries the data required to register a reusable credential.
type AddCredentialInput struct {
	Label    string
	CertPath string
}

// UpdateCredentialPathInput updates the PKCS#12 path of an existing credential.
type UpdateCredentialPathInput struct {
	CredentialID string
	CertPath     string
}

// UpdateCredentialDataInput carries data to update a credential's label.
type UpdateCredentialDataInput struct {
	CredentialID string
	Label        string
}

// storeInterface defines the DB operations required by the Manager.
type storeInterface interface {
	CreateCredential(ctx context.Context, c *nfse.Credential) error
	ListCredentials(ctx context.Context) ([]nfse.Credential, error)
	CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error)
	UpdateCredential(ctx context.Context, c *nfse.Credential) error
}

// Manager owns the credential use cases.
type Manager struct {
	store storeInterface
}

func NewManager(store storeInterface) *Manager {
	return &Manager{
		store: store,
	}
}

// AddCredential registers a reusable credential record.
func (m *Manager) AddCredential(ctx context.Context, input AddCredentialInput) error {
	if err := validateCertificatePath(input.CertPath); err != nil {
		return err
	}
	credential := &nfse.Credential{
		ID:       nfse.CredentialID(nfse.GenerateID()),
		Label:    input.Label,
		CertPath: input.CertPath,
	}
	if credential.Label == "" {
		credential.Label = input.CertPath
	}

	if err := m.store.CreateCredential(ctx, credential); err != nil {
		return fmt.Errorf("salvar credencial: %w", err)
	}
	return nil
}

// ListCredentials returns all reusable credentials.
func (m *Manager) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	credentials, err := m.store.ListCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar credenciais: %w", err)
	}
	return credentials, nil
}

// UpdateCredentialPath updates the PKCS#12 path of an existing credential.
func (m *Manager) UpdateCredentialPath(ctx context.Context, input UpdateCredentialPathInput) error {
	if err := validateCertificatePath(input.CertPath); err != nil {
		return err
	}
	cred, err := m.store.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return fmt.Errorf("credencial não encontrada: %w", err)
	}
	cred.CertPath = input.CertPath
	if err := m.store.UpdateCredential(ctx, cred); err != nil {
		return fmt.Errorf("atualizar credencial: %w", err)
	}
	return nil
}

// UpdateCredentialData updates the label of an existing credential.
func (m *Manager) UpdateCredentialData(ctx context.Context, input UpdateCredentialDataInput) error {
	cred, err := m.store.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return fmt.Errorf("credencial não encontrada: %w", err)
	}

	cred.Label = input.Label

	if err := m.store.UpdateCredential(ctx, cred); err != nil {
		return fmt.Errorf("atualizar credencial: %w", err)
	}
	return nil
}

func validateCertificatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo de certificado não encontrado: %s", path)
		}
		return fmt.Errorf("verificar certificado: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("caminho do certificado aponta para um diretório: %s", path)
	}
	return nil
}
