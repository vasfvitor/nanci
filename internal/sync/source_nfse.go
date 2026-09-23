package sync

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const requestDelay = 500 * time.Millisecond

// syncRequestDelay is a var so tests can drop the pause between ADN requests.
var syncRequestDelay = requestDelay

// nfsePayloadLimits bounds the decoded size of one ADN document.
var nfsePayloadLimits = gzipxml.Limits{
	CompressedBytes:   5 * 1024 * 1024,
	UncompressedBytes: 20 * 1024 * 1024,
}

type documentFetcher interface {
	FetchDocuments(ctx context.Context, req adn.DistributionRequest) (*adn.DocumentResponse, error)
}

// nfseSource walks the NFS-e distribution of the Ambiente de Dados Nacional (ADN).
type nfseSource struct {
	fetcher documentFetcher
	store   *Store
	xml     files.XMLStore
	log     *slog.Logger
}

// NewNFSeSource returns the Source for the ADN NFS-e distribution.
func NewNFSeSource(fetcher documentFetcher, store *Store, xml files.XMLStore, log *slog.Logger) Source {
	return &nfseSource{
		fetcher: fetcher,
		store:   store,
		xml:     xml,
		log:     log,
	}
}

func (s *nfseSource) Kind() nfse.SyncSource {
	return nfse.SyncSourceNFSe
}

func (s *nfseSource) Policy() SourcePolicy {
	return SourcePolicy{RequestDelay: syncRequestDelay}
}

// Fetch asks the ADN for the documents after cursor. The ADN has no "caught
// up" signal besides an empty batch, so an empty batch ends the run.
func (s *nfseSource) Fetch(ctx context.Context, company *nfse.Company, cursor int64) (Batch, error) {
	resp, err := s.fetcher.FetchDocuments(ctx, adn.DistributionRequest{
		LastNSU:          cursor,
		ConsultationCNPJ: company.CNPJ,
	})
	if err != nil {
		return Batch{}, err
	}

	batch := Batch{
		Items:      make([]Item, 0, len(resp.Docs)),
		UltNSU:     resp.UltNSU,
		MaxNSU:     resp.MaxNSU,
		NextCursor: cursor,
	}
	for _, env := range resp.Docs {
		var attrs []slog.Attr
		if env.DocumentType != "" {
			attrs = append(attrs, slog.String("tipo_documento", env.DocumentType))
		}
		if env.EventType != "" {
			attrs = append(attrs, slog.String("tipo_evento", env.EventType))
		}
		batch.Items = append(batch.Items, Item{
			NSU:      env.NSU,
			Schema:   env.Schema,
			Payload:  env.PayloadBase64(),
			IsEvent:  env.IsEvent(),
			LogAttrs: attrs,
		})
		batch.NextCursor = max(batch.NextCursor, env.NSU)
	}
	slices.SortFunc(batch.Items, func(a, b Item) int {
		return cmp.Compare(a.NSU, b.NSU)
	})

	if len(batch.Items) == 0 {
		batch.Done = true
		batch.StopReason = nfse.SyncStopReasonEmptyLimit
	}
	return batch, nil
}

func (s *nfseSource) ProcessItem(ctx context.Context, company *nfse.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	if item.IsEvent {
		return s.processEvent(ctx, company, item, commit)
	}
	return s.processDocument(ctx, company, src, item, commit)
}

// processDocument decodes, parses and saves a single document.
func (s *nfseSource) processDocument(ctx context.Context, company *nfse.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	s.log.Log(ctx, slog.Level(-8), "Processando documento", slog.Int64("nsu", item.NSU))

	payload, err := gzipxml.Decode(item.Payload, nfsePayloadLimits)
	if err != nil {
		return ItemOutcome{}, &ProcessingError{Op: "decode document", NSU: item.NSU, Err: err}
	}

	doc, _, err := nfse.ParseDocumentXML(payload.XML)
	if err != nil {
		return ItemOutcome{}, &ProcessingError{
			Op:         "parse document",
			NSU:        item.NSU,
			Schema:     item.Schema,
			Attrs:      item.LogAttrs,
			XMLPreview: xmlPreview(payload.XML),
			RawHash:    keepUnparsedXML(ctx, s.xml, s.log, payload),
			Err:        err,
		}
	}

	if shouldSkipDocumentByInitialPolicy(company, src, doc.IssueDate) {
		outcome, err := commitSkip(ctx, commit, false)
		if err != nil {
			return ItemOutcome{}, err
		}
		s.log.InfoContext(ctx, "Documento descartado pela política inicial de histórico",
			slog.Int64("nsu", item.NSU),
			slog.Time("issue_date", doc.IssueDate))
		return outcome, nil
	}

	doc.ID = nfse.DocumentID(uuid.NewString())
	doc.RawHash = payload.SHA256

	if err := s.xml.Store(doc.RawHash, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	doc.XMLPath = doc.RawHash + ".xml"

	params := nfse.ApplyDocumentParams{
		Document:      doc,
		Participation: nfse.ClassifyCompanyParticipation(&doc, company.CNPJ),
		CompanyID:     company.ID,
		NSU:           item.NSU,
	}
	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		applied, err := s.store.doApplyDocument(ctx, tx, s.store.queries.WithTx(tx), params)
		return ItemOutcome{Inserted: applied.Inserted}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply document failed: %w", err)
	}
	return outcome, nil
}

// processEvent decodes and saves an event. Events whose document is not
// stored for the company are skipped by policy.
func (s *nfseSource) processEvent(ctx context.Context, company *nfse.Company, item Item, commit CommitFunc) (ItemOutcome, error) {
	s.log.Log(ctx, slog.Level(-8), "Processando evento", slog.Int64("nsu", item.NSU))

	payload, err := gzipxml.Decode(item.Payload, nfsePayloadLimits)
	if err != nil {
		return ItemOutcome{}, &ProcessingError{Op: "decode event", NSU: item.NSU, Err: err}
	}

	ev, _, err := nfse.ParseEventXML(payload.XML)
	if err != nil {
		return ItemOutcome{}, &ProcessingError{
			Op:         "parse event",
			NSU:        item.NSU,
			Schema:     item.Schema,
			Attrs:      item.LogAttrs,
			XMLPreview: xmlPreview(payload.XML),
			RawHash:    keepUnparsedXML(ctx, s.xml, s.log, payload),
			Err:        err,
		}
	}

	hasLocalDocument, err := s.store.CompanyDocumentExistsByAccessKey(ctx, company.ID, string(ev.ChaveAcesso))
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("check local document for event failed: %w", err)
	}
	if !hasLocalDocument {
		outcome, err := commitSkip(ctx, commit, true)
		if err != nil {
			return ItemOutcome{}, err
		}
		s.log.InfoContext(ctx, "Evento descartado por não possuir documento local correspondente",
			slog.Int64("nsu", item.NSU),
			slog.String("chave", string(ev.ChaveAcesso)))
		return outcome, nil
	}

	ev.ID = nfse.GenerateID()
	ev.RawHash = payload.SHA256

	if err := s.xml.Store(ev.RawHash, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("event file save failed: %w", err)
	}
	ev.RawXMLPath = ev.RawHash + ".xml"

	params := nfse.ApplyEventParams{
		Event:     ev,
		CompanyID: company.ID,
		NSU:       item.NSU,
	}
	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		applied, err := s.store.doApplyEvent(ctx, tx, s.store.queries.WithTx(tx), params)
		return ItemOutcome{Inserted: applied.Inserted, IsEvent: true}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply event failed: %w", err)
	}
	return outcome, nil
}
