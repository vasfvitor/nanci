package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// dfeItems holds the item steps the NF-e and CT-e sources share. Both
// distributions deliver docZip items of the same shape: the source picks the
// parser for the schema, checks tpAmb and applies the parsed item through its
// repository; the steps here decode the payload, keep the XML, run the policy
// for an event of an unknown chave and commit the item.
type dfeItems struct {
	label string // NF-e or CT-e, for logs and errors
	xml   files.XMLStore
	log   *slog.Logger
	// documentExists reports whether the company already sees the chave.
	documentExists func(ctx context.Context, companyID dfe.CompanyID, chave string) (bool, error)
}

// applyFunc stores a parsed item through tx. rawHash is the hash of its kept
// XML.
type applyFunc func(tx *sql.Tx, rawHash string) (inserted bool, err error)

// decode decodes the item payload and returns it with the tpAmb of the pull.
func (s dfeItems) decode(ctx context.Context, company *company.Company, item Item) (gzipxml.Decoded, string, error) {
	s.log.Log(ctx, logger.LevelTrace, "Processando documento "+s.label, slog.Int64("nsu", item.NSU), slog.String("schema", item.Schema))

	payload, err := gzipxml.Decode(item.Payload, dfePayloadLimits)
	if err != nil {
		return gzipxml.Decoded{}, "", &ProcessingError{Op: "decode document", NSU: item.NSU, Schema: item.Schema, Err: err}
	}
	tpAmb, err := sefaz.TpAmb(company.Environment)
	if err != nil {
		return gzipxml.Decoded{}, "", err
	}
	return payload, tpAmb, nil
}

// keepDocument keeps the XML of a parsed document and applies it in the item
// transaction. partial marks a resumo.
func (s dfeItems) keepDocument(ctx context.Context, payload gzipxml.Decoded, commit CommitFunc, partial bool, apply applyFunc) (ItemOutcome, error) {
	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		inserted, err := apply(tx, payload.SHA256)
		return ItemOutcome{Inserted: inserted, Partial: partial}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply %s document failed: %w", s.label, err)
	}
	return outcome, nil
}

// dfeEvent is what keepEvent reads of a parsed event.
type dfeEvent struct {
	chaveAcesso string
	autorCNPJ   string
	tpEvento    string
}

// keepEvent stores an event. An event for a chave the company does not see
// yet is skipped by policy, unless the company authored it: that one is kept
// and linked when the document arrives.
func (s dfeItems) keepEvent(ctx context.Context, company *company.Company, item Item, payload gzipxml.Decoded, commit CommitFunc, ev dfeEvent, apply applyFunc) (ItemOutcome, error) {
	authoredByCompany := cnpj.Clean(ev.autorCNPJ) == company.CNPJ
	if !authoredByCompany {
		hasLocalDocument, err := s.documentExists(ctx, company.ID, ev.chaveAcesso)
		if err != nil {
			return ItemOutcome{}, fmt.Errorf("check local document for event failed: %w", err)
		}
		if !hasLocalDocument {
			outcome, err := commitSkip(ctx, commit, true)
			if err != nil {
				return ItemOutcome{}, err
			}
			s.log.InfoContext(ctx, "Evento "+s.label+" descartado por não possuir documento local correspondente",
				slog.Int64("nsu", item.NSU),
				slog.String("tp_evento", ev.tpEvento))
			return outcome, nil
		}
	}

	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("event file save failed: %w", err)
	}
	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		inserted, err := apply(tx, payload.SHA256)
		return ItemOutcome{Inserted: inserted, IsEvent: true}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply %s event failed: %w", s.label, err)
	}
	return outcome, nil
}

// keepUnsupported keeps the XML of a schema nanci does not read and lets the
// cursor move past it.
func (s dfeItems) keepUnsupported(ctx context.Context, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	outcome, err := commit(ctx, func(*sql.Tx) (ItemOutcome, error) {
		return ItemOutcome{Unsupported: true}, nil
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("persist unsupported item progress failed: %w", err)
	}
	s.log.WarnContext(ctx, "Documento "+s.label+" com schema não suportado ignorado",
		slog.Int64("nsu", item.NSU),
		slog.String("schema", item.Schema),
		slog.String("raw_hash", payload.SHA256))
	return outcome, nil
}

// parseError keeps the XML that failed to parse and wraps err so the loop
// can skip the item after repeated failures.
func (s dfeItems) parseError(ctx context.Context, op string, item Item, payload gzipxml.Decoded, err error) error {
	return &ProcessingError{
		Op:         op,
		NSU:        item.NSU,
		Schema:     item.Schema,
		XMLPreview: xmlPreview(payload.XML),
		RawHash:    keepUnparsedXML(ctx, s.xml, s.log, payload),
		Err:        err,
	}
}
