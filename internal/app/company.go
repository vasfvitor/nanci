package app

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// AddCompanyInput carries the data required to register a new company.
type AddCompanyInput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	CertPath        string
	Environment     string // "producao" | "producao_restrita"
}

// AddCompany registers a new company in the store.
func (a *App) AddCompany(ctx context.Context, input AddCompanyInput) error {
	if err := cnpj.Validate(input.CNPJ); err != nil {
		return fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(input.CNPJ)
	root, _ := cnpj.Root(cleanedCNPJ)

	credentialID, err := a.resolveCredentialForCompany(ctx, input)
	if err != nil {
		return err
	}

	company := &nfse.Company{
		ID:           uuid.NewString(),
		CNPJ:         cleanedCNPJ,
		CNPJRoot:     root,
		Name:         input.Name,
		CredentialID: credentialID,
	}

	if err := a.Store.CreateCompany(ctx, company); err != nil {
		return fmt.Errorf("salvar empresa: %w", err)
	}

	return nil
}

// ListCompanies returns all registered companies.
func (a *App) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	companies, err := a.Store.ListCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar empresas: %w", err)
	}
	return companies, nil
}

// AssignCredentialToCompany changes the active credential for an existing company.
func (a *App) AssignCredentialToCompany(ctx context.Context, input AssignCredentialInput) error {
	if err := cnpj.Validate(input.CompanyCNPJ); err != nil {
		return fmt.Errorf("CNPJ inválido: %w", err)
	}
	cleanedCNPJ := cnpj.Clean(input.CompanyCNPJ)

	company, err := a.Store.GetCompany(ctx, cleanedCNPJ)
	if err != nil {
		return fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	credential, err := a.Store.GetCredential(ctx, input.CredentialID)
	if err != nil {
		return fmt.Errorf("buscar credencial: %w", err)
	}
	if credential == nil {
		return fmt.Errorf("credencial não encontrada")
	}

	if company.CNPJRoot != "" && credential.OwnerCNPJRoot != "" && company.CNPJRoot != credential.OwnerCNPJRoot {
		return fmt.Errorf("a credencial informada não pertence à mesma raiz do CNPJ da empresa")
	}

	if err := a.Store.AssignCredentialToCompany(ctx, company.ID, credential.ID); err != nil {
		return fmt.Errorf("atribuir credencial: %w", err)
	}
	return nil
}

func (a *App) resolveCredentialForCompany(ctx context.Context, input AddCompanyInput) (string, error) {
	if input.CredentialID != "" {
		credential, err := a.Store.GetCredential(ctx, input.CredentialID)
		if err != nil {
			return "", fmt.Errorf("buscar credencial: %w", err)
		}
		if credential == nil {
			return "", fmt.Errorf("credencial não encontrada")
		}
		return credential.ID, nil
	}

	if _, err := os.Stat(input.CertPath); os.IsNotExist(err) {
		return "", fmt.Errorf("arquivo de certificado não encontrado: %s", input.CertPath)
	}

	credential := &nfse.Credential{
		ID:          uuid.NewString(),
		Label:       input.CredentialLabel,
		CertPath:    input.CertPath,
		Environment: input.Environment,
	}
	if credential.Label == "" {
		if input.Name != "" {
			credential.Label = input.Name
		} else {
			credential.Label = input.CertPath
		}
	}

	if err := a.Store.CreateCredential(ctx, credential); err != nil {
		return "", fmt.Errorf("salvar credencial: %w", err)
	}
	return credential.ID, nil
}
