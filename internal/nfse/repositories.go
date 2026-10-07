package nfse

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
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
	ProgressParams syncstate.PersistSyncProgressParams
}
