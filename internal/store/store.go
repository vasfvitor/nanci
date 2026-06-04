package store

import (
	"context"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// DocumentFilter represents the available filters for listing documents.
type DocumentFilter struct {
	Competence string // Format YYYY-MM
	Direction  string // "tomada" | "prestada" | "intermediario"
	Status     string // "normal" | "cancelada" | "substituida"
}

// CompanyStats contains summarized statistics for a company.
type CompanyStats struct {
	TotalDocuments int
	TotalTomadas   int
	TotalPrestadas int
	TotalCanceled  int
	LastNSU        int64
}

// Store defines the contract for data persistence.
// It can be implemented with SQLite (MVP) or other technologies.
type Store interface {
	// Companies
	CreateCompany(ctx context.Context, c *nfse.Company) error
	GetCompany(ctx context.Context, cnpj string) (*nfse.Company, error)
	ListCompanies(ctx context.Context) ([]nfse.Company, error)
	UpdateLastNSU(ctx context.Context, companyID string, nsu int64) error

	// Documents
	SaveDocument(ctx context.Context, doc *nfse.Document) error
	GetDocumentByChave(ctx context.Context, chave string) (*nfse.Document, error)
	ListDocuments(ctx context.Context, companyID string, filter DocumentFilter) ([]nfse.Document, error)
	GetCompanyStats(ctx context.Context, companyID string) (*CompanyStats, error)

	// Events
	SaveEvent(ctx context.Context, event *nfse.Event) error
	ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error)

	// Sync Runs
	CreateSyncRun(ctx context.Context, run *nfse.SyncRun) error
	UpdateSyncRun(ctx context.Context, run *nfse.SyncRun) error
}
