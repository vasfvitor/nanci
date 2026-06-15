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
	Environment   string
	LastNSU       int64
	TotalEmitidas int64
	TotalTomadas  int64
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

	var totalEmitidas, totalTomadas int64
	query := `
		SELECT company_role, COUNT(*) 
		FROM company_documents 
		WHERE company_id = ? 
		GROUP BY company_role
	`
	rows, err := a.DB.QueryContext(ctx, query, string(company.ID))
	if err != nil {
		return StatusResult{}, fmt.Errorf("contar documentos: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		var count int64
		if err := rows.Scan(&role, &count); err != nil {
			return StatusResult{}, fmt.Errorf("ler contagem: %w", err)
		}
		if role == "prestada" {
			totalEmitidas = count
		} else if role == "tomada" {
			totalTomadas = count
		}
	}

	return StatusResult{
		CompanyName:   company.Name,
		CNPJ:          company.CNPJ,
		Environment:   string(company.Environment),
		LastNSU:       company.LastNSU,
		TotalEmitidas: totalEmitidas,
		TotalTomadas:  totalTomadas,
	}, nil
}
