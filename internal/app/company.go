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
	CNPJ        string
	Name        string
	CertPath    string
	Environment string // "producao" | "producao_restrita"
}

// AddCompany registers a new company in the store.
func (a *App) AddCompany(ctx context.Context, input AddCompanyInput) error {
	if err := cnpj.Validate(input.CNPJ); err != nil {
		return fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(input.CNPJ)
	root, _ := cnpj.Root(cleanedCNPJ)

	if _, err := os.Stat(input.CertPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de certificado não encontrado: %s", input.CertPath)
	}

	company := &nfse.Company{
		ID:          uuid.NewString(),
		CNPJ:        cleanedCNPJ,
		CNPJRoot:    root,
		Name:        input.Name,
		CertPath:    input.CertPath,
		Environment: input.Environment,
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
