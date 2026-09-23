package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/foundation/redact"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// Item is one distributed document or event, still base64+gzip encoded.
type Item struct {
	NSU     int64
	Schema  string // ADN "Schema" or SEFAZ docZip@schema, e.g. "resNFe_v1.01.xsd"
	Payload string // base64 of the gzipped XML
	IsEvent bool
	// LogAttrs are source details reported with a processing error, such as
	// the ADN tipo_documento.
	LogAttrs []slog.Attr
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
	// Partial is set when the item stored a summary of a document rather
	// than the document itself (an NF-e resumo).
	Partial bool
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

// commitSkip commits an item skipped by policy so the checkpoint moves past it.
func commitSkip(ctx context.Context, commit CommitFunc, isEvent bool) (ItemOutcome, error) {
	outcome, err := commit(ctx, func(*sql.Tx) (ItemOutcome, error) {
		return ItemOutcome{SkippedByPolicy: true, IsEvent: isEvent}, nil
	})
	if err != nil {
		what := "document"
		if isEvent {
			what = "event"
		}
		return ItemOutcome{}, fmt.Errorf("persist skipped %s progress failed: %w", what, err)
	}
	return outcome, nil
}

// keepUnparsedXML saves the XML of an item that failed to parse, so it can
// be inspected once the loop gives up on it. It returns the blob hash, or ""
// when the save failed.
func keepUnparsedXML(ctx context.Context, xml files.XMLStore, log *slog.Logger, payload gzipxml.Decoded) string {
	if err := xml.Store(payload.SHA256, payload.XML); err != nil {
		log.WarnContext(ctx, "Falha ao salvar XML não interpretado", slog.Any("err", err))
		return ""
	}
	return payload.SHA256
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

// xmlPreview is the start of data for error messages and logs, with the
// identifiers of companies, people and documents masked. It masks the whole
// document first so the cut cannot leave half an element unmasked.
func xmlPreview(data []byte) string {
	preview := strings.Join(strings.Fields(string(redact.MaskXMLIdentifiers(data))), " ")
	if len(preview) > 400 {
		return preview[:400] + "...(truncated)"
	}
	return preview
}
