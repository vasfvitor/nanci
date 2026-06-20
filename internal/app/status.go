package app

import (
	"context"
	"fmt"
	"time"
)

// StatusResult holds the display-ready information about a company's sync state.
type StatusResult struct {
	CompanyName        string
	CNPJ               string
	Environment        string
	ConsultationCNPJ   string
	CredentialCNPJ     string
	CredentialNotAfter *time.Time
	LastCheckedNSU     int64
	LastFoundNSU       int64
	LastFoundNSUValid  bool
	LastSyncAt         *time.Time
	LastRunStatus      string
	LastRunStopReason  string
	TotalEmitidas      int64
	TotalTomadas       int64
}

// Status returns the current synchronisation state of the given company.
func (a *App) Status(ctx context.Context, rawCNPJ string) (StatusResult, error) {
	company, err := a.companyByCNPJ(ctx, rawCNPJ)
	if err != nil {
		return StatusResult{}, err
	}
	credential, err := a.credentialByID(ctx, company.CredentialID)
	if err != nil {
		return StatusResult{}, err
	}
	snapshot, err := a.SyncRepo.LatestSyncSnapshot(ctx, company.ID, company.Environment, company.CNPJ)
	if err != nil {
		return StatusResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
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
	defer func() { _ = rows.Close() }()

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

	result := StatusResult{
		CompanyName:        company.Name,
		CNPJ:               company.CNPJ,
		Environment:        string(company.Environment),
		ConsultationCNPJ:   company.CNPJ,
		CredentialCNPJ:     credential.OwnerCNPJ,
		CredentialNotAfter: credential.NotAfter,
		TotalEmitidas:      totalEmitidas,
		TotalTomadas:       totalTomadas,
	}
	if snapshot.State != nil {
		result.LastCheckedNSU = snapshot.State.LastCheckedNSU
		result.LastFoundNSU = snapshot.State.LastFoundNSU
		result.LastFoundNSUValid = snapshot.State.LastFoundNSUValid
		if snapshot.State.LastSuccessAt != nil {
			result.LastSyncAt = snapshot.State.LastSuccessAt
		}
	}
	if snapshot.Run != nil {
		result.LastRunStatus = string(snapshot.Run.Status)
		result.LastRunStopReason = string(snapshot.Run.StopReason)
		if snapshot.Run.FinishedAt != nil {
			result.LastSyncAt = snapshot.Run.FinishedAt
		}
	}
	return result, nil
}
