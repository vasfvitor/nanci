package app

import (
	"context"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// Facade methods for the focused services. They are one-liner delegations
// so the existing CLI/desktop call sites continue to work unchanged. New
// code should reach for the service structs (a.company, a.credential,
// a.status, …) directly.

func (a *App) AddCompany(ctx context.Context, input AddCompanyInput) error {
	return a.company.AddCompany(ctx, input)
}

func (a *App) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	return a.company.ListCompanies(ctx)
}

func (a *App) AssignCredentialToCompany(ctx context.Context, input AssignCredentialInput) error {
	return a.company.AssignCredentialToCompany(ctx, input)
}

func (a *App) UpdateCompany(ctx context.Context, input UpdateCompanyInput) error {
	return a.company.UpdateCompany(ctx, input)
}

func (a *App) AddCredential(ctx context.Context, input AddCredentialInput) error {
	return a.credential.AddCredential(ctx, input)
}

func (a *App) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	return a.credential.ListCredentials(ctx)
}

func (a *App) UpdateCredentialPath(ctx context.Context, input UpdateCredentialPathInput) error {
	return a.credential.UpdateCredentialPath(ctx, input)
}

func (a *App) UpdateCredentialData(ctx context.Context, input UpdateCredentialDataInput) error {
	return a.credential.UpdateCredentialData(ctx, input)
}

func (a *App) ListDocuments(ctx context.Context, input ListInput) ([]nfse.CompanyDocument, error) {
	return a.documents.ListDocuments(ctx, input)
}

func (a *App) MarkDocumentsViewed(ctx context.Context, input ListInput) (int, error) {
	return a.documents.MarkDocumentsViewed(ctx, input)
}

func (a *App) ListEventsForDocument(ctx context.Context, documentID string) ([]EventView, error) {
	return a.documents.ListEventsForDocument(ctx, documentID)
}

func (a *App) ExportCSV(ctx context.Context, input ExportInput) (ExportResult, error) {
	return a.exports.ExportCSV(ctx, input)
}

func (a *App) ExportXLSX(ctx context.Context, input ExportInput) (ExportResult, error) {
	return a.exports.ExportXLSX(ctx, input)
}

func (a *App) ExportZIP(ctx context.Context, input ExportInput) (ExportResult, error) {
	return a.exports.ExportZIP(ctx, input)
}

func (a *App) ExportDANFSeZIP(ctx context.Context, input ExportInput) (ExportResult, error) {
	return a.exports.ExportDANFSeZIP(ctx, input)
}

func (a *App) ExportDANFSe(ctx context.Context, input ExportDANFSeInput) error {
	return a.exports.ExportDANFSe(ctx, input)
}

func (a *App) ExportXML(ctx context.Context, input ExportXMLInput) error {
	return a.exports.ExportXML(ctx, input)
}

func (a *App) CountPendingExportDocuments(ctx context.Context, input ExportInput, kind string) (int, error) {
	return a.exports.CountPendingExportDocuments(ctx, input, kind)
}

func (a *App) Pull(ctx context.Context, input PullInput) (PullResult, error) {
	return a.sync.Pull(ctx, input)
}

func (a *App) ResetSyncState(ctx context.Context, input ResetSyncInput) error {
	return a.sync.ResetSyncState(ctx, input)
}

func (a *App) Status(ctx context.Context, rawCNPJ string) (StatusResult, error) {
	return a.status.Status(ctx, rawCNPJ)
}

func (a *App) QueryNFSeEvents(ctx context.Context, input QueryNFSeInput) (string, error) {
	return a.query.QueryNFSeEvents(ctx, input)
}

func (a *App) TestConnection(ctx context.Context, companyCNPJ string) (ConnectionTestResult, error) {
	return a.query.TestConnection(ctx, companyCNPJ)
}
