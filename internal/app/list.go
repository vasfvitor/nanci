package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// ListInput defines the filters for listing documents.
type ListInput struct {
	CNPJ       string
	Competence string // "YYYY-MM", optional
	Direction  string // "tomada" | "prestada" | "intermediario", optional
	OnlyUnread bool   // If true, returns only documents with viewed_at IS NULL
}

// DocumentService owns the document list/view use cases.
type DocumentService struct {
	CompanyRepo     CompanyRepository
	DocumentReader  DocumentReader
	DocumentTracker DocumentTracker
}

func newDocumentService(d Dependencies) DocumentService {
	return DocumentService{
		CompanyRepo:     d.CompanyRepo,
		DocumentReader:  d.DocumentReader,
		DocumentTracker: d.DocumentTracker,
	}
}

// buildFilter resolves a ListInput into an nfse.DocumentFilter, applying
// the company's sync-start policy as a date floor.
func (s DocumentService) buildFilter(ctx context.Context, input ListInput) (nfse.CompanyID, nfse.DocumentFilter, error) {
	company, err := lookupCompanyByCNPJ(ctx, s.CompanyRepo, input.CNPJ)
	if err != nil {
		return "", nfse.DocumentFilter{}, err
	}
	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	}
	if company.SyncStartPolicy != "" && company.SyncStartPolicy != nfse.SyncStartPolicyAll && company.SyncStartDate != nil {
		filter.IssueDateGTE = company.SyncStartDate
	}
	return company.ID, filter, nil
}

// ListDocuments returns the company-facing fiscal documents matching the given filters.
func (s DocumentService) ListDocuments(ctx context.Context, input ListInput) ([]nfse.CompanyDocument, error) {
	companyID, filter, err := s.buildFilter(ctx, input)
	if err != nil {
		return nil, err
	}
	docs, err := s.DocumentReader.ListCompanyDocuments(ctx, companyID, filter)
	if err != nil {
		return nil, fmt.Errorf("listar documentos: %w", err)
	}
	return docs, nil
}

// MarkDocumentsViewed marks documents matching the given filters as viewed.
// Returns the number of documents updated.
func (s DocumentService) MarkDocumentsViewed(ctx context.Context, input ListInput) (int, error) {
	companyID, filter, err := s.buildFilter(ctx, input)
	if err != nil {
		return 0, err
	}
	count, err := s.DocumentTracker.MarkDocumentsViewed(ctx, companyID, filter)
	if err != nil {
		return 0, fmt.Errorf("marcar documentos como vistos: %w", err)
	}
	return count, nil
}
