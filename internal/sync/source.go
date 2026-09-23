package sync

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// Item is one distributed document or event, still base64+gzip encoded.
type Item struct {
	NSU       int64
	Schema    string // ADN "Schema" or SEFAZ docZip@schema, e.g. "resNFe_v1.01.xsd"
	Payload   string // base64 of the gzipped XML
	IsEvent   bool
	DocType   string // ADN TipoDocumento; empty for SEFAZ
	EventType string // ADN TipoEvento; empty for SEFAZ
}

// Batch is one distribution response, already interpreted by the source.
type Batch struct {
	Items      []Item
	UltNSU     int64
	MaxNSU     int64
	NextCursor int64               // cursor for the next request (NFS-e: max item NSU; NF-e: ultNSU)
	Done       bool                // the source says: stop fetching for now
	StopReason nfse.SyncStopReason // meaningful when Done
	WaitUntil  *time.Time          // earliest next query the source allows
}

// ItemOutcome tells the loop how one item ended.
type ItemOutcome struct {
	Inserted        bool
	IsEvent         bool
	SkippedByPolicy bool
	Unsupported     bool // unknown schema: checkpointed and counted, never fails the run
	// Completeness is set when the item stored an NF-e document: resumo for
	// a resNFe, completa for a procNFe. Empty for events and NFS-e.
	Completeness nfe.Completeness
}

// CommitFunc runs write in one transaction together with the item's sync checkpoint.
type CommitFunc func(ctx context.Context, write func(tx *sql.Tx) (ItemOutcome, error)) (ItemOutcome, error)

// SourcePolicy holds the request limits of a distribution service.
type SourcePolicy struct {
	RequestDelay    time.Duration // pause between fetches
	RequestsPerHour int           // 0 means unlimited
}

// Source is one distribution service the pull loop can walk by NSU.
type Source interface {
	Kind() nfse.SyncSource
	Policy() SourcePolicy
	Fetch(ctx context.Context, company *nfse.Company, cursor int64) (Batch, error)
	// ProcessItem decodes, parses and stores the raw XML outside any
	// transaction, then calls commit exactly once, also for policy skips, so
	// the checkpoint advances. Decode and parse failures are returned as
	// *ProcessingError.
	ProcessItem(ctx context.Context, company *nfse.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error)
}

// shouldSkipDocumentByInitialPolicy reports whether a document issued at
// issueDate falls before the company start date while the source has not
// finished its initial sync yet.
func shouldSkipDocumentByInitialPolicy(company *nfse.Company, src SourceState, issueDate time.Time) bool {
	if src.InitialSyncDoneAt != nil {
		return false
	}
	if company.SyncStartPolicy == "" || company.SyncStartPolicy == nfse.SyncStartPolicyAll {
		return false
	}
	if company.SyncStartDate == nil {
		return false
	}
	return issueDate.Format("2006-01-02") < company.SyncStartDate.Format("2006-01-02")
}

func xmlPreview(data []byte) string {
	preview := strings.TrimSpace(string(data))
	preview = strings.ReplaceAll(preview, "\r", " ")
	preview = strings.ReplaceAll(preview, "\n", " ")
	preview = strings.ReplaceAll(preview, "\t", " ")
	preview = strings.Join(strings.Fields(preview), " ")
	if len(preview) > 400 {
		return preview[:400] + "...(truncated)"
	}
	return preview
}
