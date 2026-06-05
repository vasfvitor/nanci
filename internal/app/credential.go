package app

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// AddCredentialInput carries the data required to register a reusable credential.
type AddCredentialInput struct {
	Label       string
	CertPath    string
	Environment string
}

// UpdateCredentialPathInput updates the PKCS#12 path of an existing credential.
type UpdateCredentialPathInput struct {
	CredentialID string
	CertPath     string
}

// AssignCredentialInput changes the active credential of a company.
type AssignCredentialInput struct {
	CompanyCNPJ  string
	CredentialID string
}

// AddCredential registers a reusable credential record.
func (a *App) AddCredential(ctx context.Context, input AddCredentialInput) error {
	if _, err := os.Stat(input.CertPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de certificado não encontrado: %s", input.CertPath)
	}

	credential := &nfse.Credential{
		ID:          uuid.NewString(),
		Label:       input.Label,
		CertPath:    input.CertPath,
		Environment: input.Environment,
	}
	if credential.Label == "" {
		credential.Label = input.CertPath
	}

	if err := a.Store.CreateCredential(ctx, credential); err != nil {
		return fmt.Errorf("salvar credencial: %w", err)
	}
	return nil
}

// ListCredentials returns all reusable credentials.
func (a *App) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	credentials, err := a.Store.ListCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar credenciais: %w", err)
	}
	return credentials, nil
}

// UpdateCredentialPath updates the PKCS#12 path of an existing credential.
func (a *App) UpdateCredentialPath(ctx context.Context, input UpdateCredentialPathInput) error {
	if _, err := os.Stat(input.CertPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de certificado não encontrado: %s", input.CertPath)
	}
	if err := a.Store.UpdateCredentialPath(ctx, input.CredentialID, input.CertPath); err != nil {
		return fmt.Errorf("atualizar credencial: %w", err)
	}
	return nil
}
