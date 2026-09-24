package nfse

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

type DocumentFilter struct {
	Competence   string
	Direction    string
	Status       string
	FromNSU      *int64
	ToNSU        *int64
	Limit        *int
	OnlyUnread   bool
	IssueDateGTE *time.Time
	ChavesAcesso []string
}

type DocumentExportMark struct {
	DocumentID string
	ExportKind string
	Hash       string
}

type StartRunParams struct {
	CompanyID         dfe.CompanyID
	Source            SyncSource
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
	CompanyID        dfe.CompanyID
	Source           SyncSource
	Environment      Environment
	ConsultationCNPJ string
}

type ApplyDocumentParams struct {
	Document      Document
	Participation CompanyParticipation
	CompanyID     dfe.CompanyID
	NSU           int64
}

type ApplyEventParams struct {
	Event     Event
	CompanyID dfe.CompanyID
	NSU       int64
}

type ApplyOutcome struct {
	Inserted bool
}

type ApplyDocumentAndProgressParams struct {
	DocumentParams ApplyDocumentParams
	ProgressParams PersistSyncProgressParams
}

type PersistSyncProgressParams struct {
	CompanyID             dfe.CompanyID
	Source                SyncSource
	RunID                 SyncRunID
	Environment           Environment
	ConsultationCNPJ      string
	LastProcessedNSU      int64
	LastFoundNSU          *int64
	MaxNSU                *int64 // highest NSU the source reports; nil keeps the stored value
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
	LastFoundNSU          *int64
}

type SyncSnapshot struct {
	State *SyncState
	Run   *SyncRun
}

type ResetSyncStateParams struct {
	CompanyID dfe.CompanyID
	Source    SyncSource
}

// HasSyncStateParams asks whether the company has a sync cursor for Source.
type HasSyncStateParams struct {
	CompanyID dfe.CompanyID
	Source    SyncSource
}
