package app

import (
	"context"
	"fmt"
	"os"

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

	credential, err := a.resolveCredentialForCompany(ctx, input)
	if err != nil {
		return err
	}

	company := &nfse.Company{
		ID:                 nfse.CompanyID(nfse.GenerateID()),
		CNPJ:               cleanedCNPJ,
		CNPJRoot:           root,
		Name:               input.Name,
		CredentialID:       credential.ID,
		CredentialLabel:    credential.Label,
		CredentialCertPath: credential.CertPath,
		Environment:        credential.Environment,
	}

	if err := a.CompanyRepo.CreateCompany(ctx, company); err != nil {
		return fmt.Errorf("salvar empresa: %w", err)
	}

	return nil
}

// ListCompanies returns all registered companies.
func (a *App) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	companies, err := a.CompanyRepo.ListCompanies(ctx)
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

	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	credential, err := a.CredentialRepo.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return fmt.Errorf("buscar credencial: %w", err)
	}
	if credential == nil {
		return fmt.Errorf("credencial não encontrada")
	}

	if company.CNPJRoot != "" && credential.OwnerCNPJRoot != "" && company.CNPJRoot != credential.OwnerCNPJRoot {
		return fmt.Errorf("a credencial informada não pertence à mesma raiz do CNPJ da empresa")
	}

	if err := a.CompanyRepo.AssignCredential(ctx, company.ID, credential.ID); err != nil {
		return fmt.Errorf("atribuir credencial: %w", err)
	}
	return nil
}

func (a *App) resolveCredentialForCompany(ctx context.Context, input AddCompanyInput) (*nfse.Credential, error) {
	if input.CredentialID != "" {
		credential, err := a.CredentialRepo.CredentialByID(ctx, nfse.CredentialID(input.CredentialID))
		if err != nil {
			return nil, fmt.Errorf("buscar credencial: %w", err)
		}
		if credential == nil {
			return nil, fmt.Errorf("credencial não encontrada")
		}
		return credential, nil
	}

	if _, err := os.Stat(input.CertPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("arquivo de certificado não encontrado: %s", input.CertPath)
	} else if err != nil {
		return nil, fmt.Errorf("verificar certificado: %w", err)
	}

	environment, err := nfse.ParseEnvironment(input.Environment)
	if err != nil {
		return nil, fmt.Errorf("ambiente inválido: %w", err)
	}

	credential := &nfse.Credential{
		ID:          nfse.CredentialID(nfse.GenerateID()),
		Label:       input.CredentialLabel,
		CertPath:    input.CertPath,
		Environment: environment,
	}
	if credential.Label == "" {
		if input.Name != "" {
			credential.Label = input.Name
		} else {
			credential.Label = input.CertPath
		}
	}

	if err := a.CredentialRepo.CreateCredential(ctx, credential); err != nil {
		return nil, fmt.Errorf("salvar credencial: %w", err)
	}
	return credential, nil
}

// UpdateCompanyInput carries data to update a company
type UpdateCompanyInput struct {
	CNPJ        string
	Name        string
	Environment string // "producao" | "producao_restrita"
}

// UpdateCompany updates the name and environment of an existing company.
func (a *App) UpdateCompany(ctx context.Context, input UpdateCompanyInput) error {
	cleanedCNPJ := cnpj.Clean(input.CNPJ)

	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	environment, err := nfse.ParseEnvironment(input.Environment)
	if err != nil {
		return fmt.Errorf("ambiente inválido: %w", err)
	}

	if err := a.CompanyRepo.UpdateCompany(ctx, company.ID, input.Name, environment); err != nil {
		return fmt.Errorf("atualizar empresa: %w", err)
	}

	return nil
}
