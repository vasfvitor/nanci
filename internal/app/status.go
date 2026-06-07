package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// StatusResult holds the display-ready information about a company's sync state.
type StatusResult struct {
	CompanyName string
	CNPJ        string
	Environment string
	LastNSU     int64
}

// Status returns the current synchronisation state of the given company.
func (a *App) Status(ctx context.Context, rawCNPJ string) (StatusResult, error) {
	if err := cnpj.Validate(rawCNPJ); err != nil {
		return StatusResult{}, fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(rawCNPJ)

	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return StatusResult{}, fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return StatusResult{}, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	return StatusResult{
		CompanyName: company.Name,
		CNPJ:        company.CNPJ,
		Environment: string(company.Environment),
		LastNSU:     company.LastNSU,
	}, nil
}
