package app

import (
	"context"
	"fmt"
	"os"

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
		ID:          nfse.CredentialID(nfse.GenerateID()),
		Label:       input.Label,
		CertPath:    input.CertPath,
		Environment: nfse.Environment(input.Environment),
	}
	if credential.Label == "" {
		credential.Label = input.CertPath
	}

	if err := a.CredentialRepo.CreateCredential(ctx, credential); err != nil {
		return fmt.Errorf("salvar credencial: %w", err)
	}
	return nil
}

// ListCredentials returns all reusable credentials.
func (a *App) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	credentials, err := a.CredentialRepo.ListCredentials(ctx)
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
	cred, err := a.CredentialRepo.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil || cred == nil {
		return fmt.Errorf("credencial não encontrada: %w", err)
	}
	cred.CertPath = input.CertPath
	if err := a.CredentialRepo.UpdateCredential(ctx, cred); err != nil {
		return fmt.Errorf("atualizar credencial: %w", err)
	}
	return nil
}
