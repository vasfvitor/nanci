package nfse

import (
	"context"
)

type CompanyRepository interface {
	CreateCompany(ctx context.Context, c *Company) error
	CompanyByCNPJ(ctx context.Context, cnpjVal string) (*Company, error)
	ListCompanies(ctx context.Context) ([]Company, error)
	AssignCredential(ctx context.Context, id CompanyID, credID CredentialID) error
	UpdateCompany(ctx context.Context, id CompanyID, name string, environment Environment) error
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
	ListEventsByDocument(ctx context.Context, docID string) ([]Event, error)
}

type StartRunParams struct {
	CompanyID         CompanyID
	CredentialID      CredentialID
	Environment       Environment
	CredentialCNPJ    string
	ConsultationCNPJ  string
	ConsultationBasis ConsultationBasis
	Mode              SyncMode
	FromNSU           int64
	ToNSU             int64
}

type GetOrCreateSyncStateParams struct {
	CompanyID        CompanyID
	Environment      Environment
	ConsultationCNPJ string
	LegacyLastNSU    int64
}

type ApplyDocumentParams struct {
	Document      Document
	Participation CompanyParticipation
	CompanyID     CompanyID
	NSU           int64
}

type ApplyEventParams struct {
	Event     Event
	CompanyID CompanyID
	NSU       int64
}

type AdvanceCheckpointParams struct {
	CompanyID CompanyID
	RunID     SyncRunID
	LastNSU   int64
}

type PersistSyncProgressParams struct {
	CompanyID             CompanyID
	RunID                 SyncRunID
	Environment           Environment
	ConsultationCNPJ      string
	LastCheckedNSU        int64
	LastFoundNSU          int64
	LastFoundNSUValid     bool
	LastEmptyStreak       int
	CheckedCount          int
	DocumentsFound        int
	EmptyCount            int
	ConsecutiveEmptyCount int
	ErrorsCount           int
	ErrorCode             string
	ErrorMessage          string
	MarkSuccess           bool
}

type FinishRunParams struct {
	RunID                 SyncRunID
	Status                SyncStatus
	StopReason            SyncStopReason
	ErrorCode             string
	ErrorMsg              string
	CheckedCount          int
	DocumentsFound        int
	EmptyCount            int
	ConsecutiveEmptyCount int
	ErrorsCount           int
	LastFoundNSU          int64
	LastFoundNSUValid     bool
}

type SyncSnapshot struct {
	State *SyncState
	Run   *SyncRun
}

type ResetSyncStateParams struct {
	CompanyID CompanyID
}

type SyncRepository interface {
	GetOrCreateState(ctx context.Context, params GetOrCreateSyncStateParams) (*SyncState, error)
	StartRun(ctx context.Context, params StartRunParams) (SyncRun, error)
	ApplyDocument(ctx context.Context, params ApplyDocumentParams) error
	ApplyEvent(ctx context.Context, params ApplyEventParams) error
	PersistProgress(ctx context.Context, params PersistSyncProgressParams) error
	FinishRun(ctx context.Context, params FinishRunParams) error
	LatestSyncSnapshot(ctx context.Context, companyID CompanyID, environment Environment, consultationCNPJ string) (SyncSnapshot, error)
	ResetSyncState(ctx context.Context, params ResetSyncStateParams) error
}
