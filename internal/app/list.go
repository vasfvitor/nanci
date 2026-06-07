package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// ListInput defines the filters for listing documents.
type ListInput struct {
	CNPJ       string
	Competence string // "YYYY-MM", optional
	Direction  string // "tomada" | "prestada" | "intermediario", optional
}

// ListDocuments returns the company-facing fiscal documents matching the given filters.
func (a *App) ListDocuments(ctx context.Context, input ListInput) ([]nfse.CompanyDocument, error) {
	if err := cnpj.Validate(input.CNPJ); err != nil {
		return nil, fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(input.CNPJ)

	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return nil, fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
	}

	docs, err := a.DocumentReader.ListCompanyDocuments(ctx, company.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("listar documentos: %w", err)
	}

	return docs, nil
}
