package app

import (
	"context"
	"fmt"
)

// StatusResult holds the display-ready information about a company's sync state.
type StatusResult struct {
	CompanyName   string
	CNPJ          string
	Environment   string
	LastNSU       int64
	TotalEmitidas int64
	TotalTomadas  int64
}

// Status returns the current synchronisation state of the given company.
func (a *App) Status(ctx context.Context, rawCNPJ string) (StatusResult, error) {
	company, err := a.companyByCNPJ(ctx, rawCNPJ)
	if err != nil {
		return StatusResult{}, err
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
		switch role {
		case "prestada":
			totalEmitidas = count
		case "tomada":
			totalTomadas = count
		}
	}

	if err := rows.Err(); err != nil {
		return StatusResult{}, fmt.Errorf("erro iterando contagem de documentos: %w", err)
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
