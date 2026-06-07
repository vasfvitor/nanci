package nfse

import (
	"context"
)

type CompanyRepository interface {
	CreateCompany(ctx context.Context, c *Company) error
	CompanyByCNPJ(ctx context.Context, cnpjVal string) (*Company, error)
	ListCompanies(ctx context.Context) ([]Company, error)
	AssignCredential(ctx context.Context, id CompanyID, credID CredentialID) error
}

type CredentialRepository interface {
	CreateCredential(ctx context.Context, c *Credential) error
	CredentialByID(ctx context.Context, id CredentialID) (*Credential, error)
	ListCredentials(ctx context.Context) ([]Credential, error)
	DeleteCredential(ctx context.Context, id CredentialID) error
	UpdateCredential(ctx context.Context, c *Credential) error
}

type DocumentFilter struct {
	Competence string
	Direction  string
	Status     string
	FromNSU    *int64
	ToNSU      *int64
	Limit      *int
}

type DocumentReader interface {
	ListCompanyDocuments(ctx context.Context, companyID CompanyID, filter DocumentFilter) ([]CompanyDocument, error)
}

type StartRunParams struct {
	CompanyID         CompanyID
	CredentialID      CredentialID
	CredentialCNPJ    string
	ConsultationCNPJ  string
	ConsultationBasis ConsultationBasis
	FromNSU           int64
	ToNSU             int64
}

type ApplyDocumentParams struct {
	Document         Document
	Participation    CompanyParticipation
	CompanyID        CompanyID
	NSU              int64
}

type ApplyEventParams struct {
	Event            Event
	CompanyID        CompanyID
	NSU              int64
}

type AdvanceCheckpointParams struct {
	CompanyID CompanyID
	RunID     SyncRunID
	LastNSU   int64
}

type FinishRunParams struct {
	RunID    SyncRunID
	Status   SyncStatus
	ErrorMsg string
}

type SyncRepository interface {
	StartRun(ctx context.Context, params StartRunParams) (SyncRun, error)
	ApplyDocument(ctx context.Context, params ApplyDocumentParams) error
	ApplyEvent(ctx context.Context, params ApplyEventParams) error
	AdvanceCheckpoint(ctx context.Context, params AdvanceCheckpointParams) error
	FinishRun(ctx context.Context, params FinishRunParams) error
}
