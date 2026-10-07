package syncstate

import "github.com/vasfvitor/nanci/internal/dfe"

type StartRunParams struct {
	CompanyID         dfe.CompanyID
	Source            SyncSource
	CredentialID      dfe.CredentialID
	Environment       dfe.Environment
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
	Environment      dfe.Environment
	ConsultationCNPJ string
}

type PersistSyncProgressParams struct {
	CompanyID             dfe.CompanyID
	Source                SyncSource
	RunID                 SyncRunID
	Environment           dfe.Environment
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
