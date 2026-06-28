package nfse

import (
	"time"
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
}

type DocumentExportMark struct {
	DocumentID string
	ExportKind string
	Hash       string
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

type ApplyOutcome struct {
	Inserted bool
}

type ApplyDocumentAndProgressParams struct {
	DocumentParams ApplyDocumentParams
	ProgressParams PersistSyncProgressParams
}

type ApplyEventAndProgressParams struct {
	EventParams    ApplyEventParams
	ProgressParams PersistSyncProgressParams
}

type PersistSyncProgressParams struct {
	CompanyID             CompanyID
	RunID                 SyncRunID
	Environment           Environment
	ConsultationCNPJ      string
	LastProcessedNSU      int64
	LastFoundNSU          *int64
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
	CompanyID CompanyID
}

type HasSyncStateParams struct {
	CompanyID CompanyID
}
