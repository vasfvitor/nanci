package main

import (
	"context"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

// The interfaces below list only the core methods App calls, so tests can
// replace each service with a small fake. servicesFrom is the one place that
// wires them to the real *app.App.

type companyService interface {
	AddCompany(ctx context.Context, input company.AddCompanyInput) error
	UpdateCompany(ctx context.Context, input company.UpdateCompanyInput) error
	CompanyByCNPJ(ctx context.Context, rawCNPJ string) (*company.Company, error)
	AssignCredentialToCompany(ctx context.Context, input company.AssignCredentialInput) error
	ListCompanies(ctx context.Context) ([]company.Company, error)
}

type credentialService interface {
	AddCredential(ctx context.Context, input credential.AddCredentialInput) error
	ListCredentials(ctx context.Context) ([]credential.Credential, error)
	UpdateCredentialPath(ctx context.Context, input credential.UpdateCredentialPathInput) error
	UpdateCredentialData(ctx context.Context, input credential.UpdateCredentialDataInput) error
}

type documentService interface {
	ListDocuments(ctx context.Context, input app.ListInput) ([]nfse.CompanyDocument, error)
	ListEventsForDocument(ctx context.Context, documentID string) ([]app.EventView, error)
	MarkDocumentsViewed(ctx context.Context, cnpj string, chaves []string) (int, error)
}

type exportService interface {
	ExportCSV(ctx context.Context, input app.ExportInput) (app.ExportResult, error)
	ExportXLSX(ctx context.Context, input app.ExportInput) (app.ExportResult, error)
	ExportZIP(ctx context.Context, input app.ExportInput) (app.ExportResult, error)
	ExportDANFSeZIP(ctx context.Context, input app.ExportInput) (app.ExportResult, error)
	ExportDANFSe(ctx context.Context, input app.ExportDANFSeInput) error
	ExportXML(ctx context.Context, input app.ExportXMLInput) error
	CountPendingExportDocuments(ctx context.Context, input app.ExportInput, kind string) (int, error)
}

type queryService interface {
	QueryNFSeEvents(ctx context.Context, input app.QueryNFSeInput) (string, error)
	TestConnection(ctx context.Context, companyCNPJ string) (app.ConnectionTestResult, error)
}

type syncService interface {
	Pull(ctx context.Context, input nsync.PullInput) (nsync.PullResult, error)
	Status(ctx context.Context, rawCNPJ string) (nsync.StatusResult, error)
	ResetSyncState(ctx context.Context, input nsync.ResetSyncInput) error
}

type nfeService interface {
	Pull(ctx context.Context, cnpj string) (app.NFePullResult, error)
	Status(ctx context.Context, cnpj string) (app.NFeStatusResult, error)
	ListDocuments(ctx context.Context, in app.NFeListInput) ([]app.NFeDocument, error)
	MarkViewed(ctx context.Context, cnpj string, chaves []string) (int, error)
	ListEvents(ctx context.Context, cnpj, chave string) ([]nfe.Event, error)
	ListPendingManifestacoes(ctx context.Context, in app.NFePendingInput) ([]app.NFePendingManifestacao, error)
	PlanCiencia(ctx context.Context, in app.NFeCienciaInput) (app.NFeCienciaPlan, error)
	RegisterCiencia(ctx context.Context, in app.NFeCienciaInput) (app.NFeManifestacaoSummary, error)
	RegisterManifestacao(ctx context.Context, in app.NFeManifestacaoInput) (app.NFeEventOutcome, error)
	ExportXML(ctx context.Context, in app.NFeExportXMLInput) error
	ExportXMLZip(ctx context.Context, in app.NFeExportInput) (app.NFeExportResult, error)
	Reset(ctx context.Context, cnpj string) (app.NFeResetResult, error)
}

type cteService interface {
	Pull(ctx context.Context, cnpj string) (app.CTePullResult, error)
	Status(ctx context.Context, cnpj string) (app.CTeStatusResult, error)
	ListDocuments(ctx context.Context, in app.ListCTeInput) ([]cte.CompanyDocument, error)
	MarkViewed(ctx context.Context, cnpj string, chaves []string) (int, error)
	ListEvents(ctx context.Context, cnpj, chave string) ([]cte.Event, error)
	TestConnection(ctx context.Context, cnpj string) (app.ConnectionTestResult, error)
	ExportXML(ctx context.Context, in app.CTeExportXMLInput) error
	ExportXMLZip(ctx context.Context, in app.CTeExportInput) (app.ExportResult, error)
	PreviewReset(ctx context.Context, cnpj string) (app.CTeResetResult, error)
	Reset(ctx context.Context, cnpj string) (app.CTeResetResult, error)
}

// services holds the core services App calls.
type services struct {
	companies   companyService
	credentials credentialService
	documents   documentService
	exports     exportService
	query       queryService
	sync        syncService
	nfe         nfeService
	cte         cteService
}

func servicesFrom(core *app.App) services {
	return services{
		companies:   core.Companies,
		credentials: core.Credentials,
		documents:   core.Documents,
		exports:     core.Exports,
		query:       core.Query,
		sync:        core.SyncManager,
		nfe:         core.NFe,
		cte:         core.CTe,
	}
}
