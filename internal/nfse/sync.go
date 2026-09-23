package nfse

import (
	"fmt"
	"time"
)

// The sync types in this file are source-neutral: they describe the NSU
// distribution cursor of any fiscal document source, not only NFS-e.

// SyncSource identifies the distribution service a sync cursor belongs to.
type SyncSource string

const (
	SyncSourceNFSe SyncSource = "nfse"
	SyncSourceNFe  SyncSource = "nfe"
	SyncSourceCTe  SyncSource = "cte"
)

func ParseSyncSource(val string) (SyncSource, error) {
	source := SyncSource(val)
	if !source.Valid() {
		return "", fmt.Errorf("invalid sync source %q: %w", val, ErrInvalidEnum)
	}
	return source, nil
}

func (s SyncSource) Valid() bool {
	switch s {
	case SyncSourceNFSe, SyncSourceNFe, SyncSourceCTe:
		return true
	default:
		return false
	}
}

func (s SyncSource) String() string {
	return string(s)
}

// SyncRun represents a synchronization execution for audit and control.
type SyncRun struct {
	ID                    SyncRunID
	CompanyID             CompanyID
	Source                SyncSource
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

// SyncState represents the persisted sync cursor and audit state for a company/source/environment/CNPJ key.
type SyncState struct {
	CompanyID        CompanyID
	Source           SyncSource
	Environment      Environment
	ConsultationCNPJ string
	LastProcessedNSU int64
	LastFoundNSU     *int64
	MaxNSU           *int64
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
	Source                   SyncSource
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
	CompletasSaved           int // NF-e procNFe stored; zero for NFS-e
	ResumosSaved             int // NF-e resNFe stored; zero for NFS-e
	DocsInBatch              int
	Errors                   int
	Message                  string
}

// ProgressFunc is a callback function to report progress.
type ProgressFunc func(event ProgressEvent)
