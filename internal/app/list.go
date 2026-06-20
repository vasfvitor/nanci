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

// ListDocuments returns the company-facing fiscal documents matching the given filters.
func (a *App) ListDocuments(ctx context.Context, input ListInput) ([]nfse.CompanyDocument, error) {
	company, err := a.companyByCNPJ(ctx, input.CNPJ)
	if err != nil {
		return nil, err
	}

	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	}

	docs, err := a.DocumentReader.ListCompanyDocuments(ctx, company.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("listar documentos: %w", err)
	}

	return docs, nil
}

// MarkDocumentsViewed marks documents matching the given filters as viewed.
// Returns the number of documents updated.
func (a *App) MarkDocumentsViewed(ctx context.Context, input ListInput) (int, error) {
	company, err := a.companyByCNPJ(ctx, input.CNPJ)
	if err != nil {
		return 0, err
	}

	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	}

	count, err := a.DocumentTracker.MarkDocumentsViewed(ctx, company.ID, filter)
	if err != nil {
		return 0, fmt.Errorf("marcar documentos como vistos: %w", err)
	}

	return count, nil
}
