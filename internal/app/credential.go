package app

import (
	"context"
	"fmt"

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

// AssignCredentialInput changes the active credential of a company.
type AssignCredentialInput struct {
	CompanyCNPJ  string
	CredentialID string
}

// CredentialService owns the credential use cases.
type CredentialService struct {
	CredentialRepo CredentialRepository
}

func newCredentialService(d Dependencies) CredentialService {
	return CredentialService{CredentialRepo: d.CredentialRepo}
}

// AddCredential registers a reusable credential record.
func (s CredentialService) AddCredential(ctx context.Context, input AddCredentialInput) error {
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

	if err := s.CredentialRepo.CreateCredential(ctx, credential); err != nil {
		return fmt.Errorf("salvar credencial: %w", err)
	}
	return nil
}

// ListCredentials returns all reusable credentials.
func (s CredentialService) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	credentials, err := s.CredentialRepo.ListCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar credenciais: %w", err)
	}
	return credentials, nil
}

// UpdateCredentialPath updates the PKCS#12 path of an existing credential.
func (s CredentialService) UpdateCredentialPath(ctx context.Context, input UpdateCredentialPathInput) error {
	if err := validateCertificatePath(input.CertPath); err != nil {
		return err
	}
	cred, err := s.CredentialRepo.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return fmt.Errorf("credencial não encontrada: %w", err)
	}
	cred.CertPath = input.CertPath
	if err := s.CredentialRepo.UpdateCredential(ctx, cred); err != nil {
		return fmt.Errorf("atualizar credencial: %w", err)
	}
	return nil
}

// UpdateCredentialDataInput carries data to update a credential's label.
type UpdateCredentialDataInput struct {
	CredentialID string
	Label        string
}

// UpdateCredentialData updates the label of an existing credential.
func (s CredentialService) UpdateCredentialData(ctx context.Context, input UpdateCredentialDataInput) error {
	cred, err := s.CredentialRepo.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return fmt.Errorf("credencial não encontrada: %w", err)
	}

	cred.Label = input.Label

	if err := s.CredentialRepo.UpdateCredential(ctx, cred); err != nil {
		return fmt.Errorf("atualizar credencial: %w", err)
	}
	return nil
}
