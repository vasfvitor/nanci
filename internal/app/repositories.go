package app

import (
	"context"

	"github.com/vasfvitor/nanci/internal/nfse"
	syncservice "github.com/vasfvitor/nanci/internal/service/sync"
)

type CompanyRepository interface {
	CreateCompany(ctx context.Context, c *nfse.Company) error
	CompanyByCNPJ(ctx context.Context, cnpjVal string) (*nfse.Company, error)
	ListCompanies(ctx context.Context) ([]nfse.Company, error)
	AssignCredential(ctx context.Context, id nfse.CompanyID, credID nfse.CredentialID) error
	UpdateCompany(ctx context.Context, c *nfse.Company) error
}

type CredentialRepository interface {
	CreateCredential(ctx context.Context, c *nfse.Credential) error
	CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error)
	ListCredentials(ctx context.Context) ([]nfse.Credential, error)
	DeleteCredential(ctx context.Context, id nfse.CredentialID) error
	UpdateCredential(ctx context.Context, c *nfse.Credential) error
}

type DocumentReader interface {
	ListCompanyDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter) ([]nfse.CompanyDocument, error)
	CompanyDocumentByChave(ctx context.Context, companyID nfse.CompanyID, chave string) (*nfse.CompanyDocument, error)
	ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error)
}

type DocumentTracker interface {
	ListPendingExportDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter, kind string) ([]nfse.CompanyDocument, error)
	CountPendingExportDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter, kind string) (int, error)
	MarkDocumentsExported(ctx context.Context, companyID nfse.CompanyID, kind string, marks []nfse.DocumentExportMark) error
	MarkDocumentsViewed(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter) (int, error)
}

type SyncRepository interface {
	syncservice.SyncRepository
	LatestSyncSnapshot(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, consultationCNPJ string) (nfse.SyncSnapshot, error)
	ResetSyncState(ctx context.Context, params nfse.ResetSyncStateParams) error
	HasSyncState(ctx context.Context, params nfse.HasSyncStateParams) (bool, error)
}
