package nfse

import (
	"time"
)

// SyncRun represents a synchronization execution for audit and control.
type SyncRun struct {
	ID                    SyncRunID
	CompanyID             CompanyID
	CredentialID          CredentialID
	Environment           Environment
	CredentialCNPJ        string
	ConsultationCNPJ      string
	ConsultationBasis     ConsultationBasis // "exact_certificate_cnpj" | "same_root_certificate"
	Mode                  SyncMode
	StartedAt             time.Time
	FinishedAt            *time.Time
	FromNSU               int64
	ToNSU                 int64
	CheckedCount          int
	DocumentsFound        int
	EmptyCount            int
	ConsecutiveEmptyCount int
	ErrorsCount           int
	LastFoundNSU          *int64
	Status                SyncStatus // "running" | "completed" | "failed" | "interrupted"
	StopReason            SyncStopReason
}

// SyncState represents the persisted sync cursor and audit state for a company/environment/CNPJ pair.
type SyncState struct {
	CompanyID        CompanyID
	Environment      Environment
	ConsultationCNPJ string
	LastProcessedNSU int64
	LastFoundNSU     *int64
	LastEmptyStreak  int
	LastSuccessAt    *time.Time
	LastErrorAt      *time.Time
	LastErrorCode    string
	LastErrorMessage string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ProgressEvent contains information about the progress of a long-running operation.
type ProgressEvent struct {
	CurrentNSU               int64
	MaxNSU                   int64
	LastProcessedNSU         int64
	LastFoundNSU             *int64
	EmptyStreak              int
	Status                   SyncStatus
	StopReason               SyncStopReason
	DocsFound                int
	DocumentsSaved           int
	EventsSaved              int
	DocumentsSkippedByPolicy int
	EventsSkippedByPolicy    int
	DocsInBatch              int
	Errors                   int
	Message                  string
}

// ProgressFunc is a callback function to report progress.
type ProgressFunc func(event ProgressEvent)
