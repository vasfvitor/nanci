package app

import (
	"context"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/store"
)

// StatusResult holds the display-ready information about a company's sync state.
type StatusResult struct {
	CompanyName        string
	CNPJ               string
	Environment        string
	ConsultationCNPJ   string
	CredentialCNPJ     string
	CredentialNotAfter *time.Time
	LastProcessedNSU   int64
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	LastRunStatus      string
	LastRunStopReason  string
	TotalEmitidas      int64
	TotalTomadas       int64
}

// SyncStatusService computes the display-ready status for a company.
type SyncStatusService struct {
	CompanyRepo    CompanyRepository
	CredentialRepo CredentialRepository
	SyncRepo       *store.SyncRepository
	DocumentReader DocumentReader
}

func newSyncStatusService(d Dependencies) SyncStatusService {
	return SyncStatusService{
		CompanyRepo:    d.CompanyRepo,
		CredentialRepo: d.CredentialRepo,
		SyncRepo:       d.SyncRepo,
		DocumentReader: d.DocumentReader,
	}
}

// Status returns the current synchronisation state of the given company.
func (s SyncStatusService) Status(ctx context.Context, rawCNPJ string) (StatusResult, error) {
	company, err := lookupCompanyByCNPJ(ctx, s.CompanyRepo, rawCNPJ)
	if err != nil {
		return StatusResult{}, err
	}
	credential, err := lookupCredentialByID(ctx, s.CredentialRepo, company.CredentialID)
	if err != nil {
		return StatusResult{}, err
	}
	snapshot, err := s.SyncRepo.LatestSyncSnapshot(ctx, company.ID, company.Environment, company.CNPJ)
	if err != nil {
		return StatusResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}

	counts, err := s.DocumentReader.CountDocumentsByRole(ctx, company.ID)
	if err != nil {
		return StatusResult{}, fmt.Errorf("contar documentos: %w", err)
	}

	result := StatusResult{
		CompanyName:        company.Name,
		CNPJ:               company.CNPJ,
		Environment:        string(company.Environment),
		ConsultationCNPJ:   company.CNPJ,
		CredentialCNPJ:     credential.OwnerCNPJ,
		CredentialNotAfter: credential.NotAfter,
		TotalEmitidas:      counts["prestada"],
		TotalTomadas:       counts["tomada"],
	}
	if snapshot.State != nil {
		result.LastProcessedNSU = snapshot.State.LastProcessedNSU
		result.LastFoundNSU = snapshot.State.LastFoundNSU
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
