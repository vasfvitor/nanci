package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
)

// nfeRequestDelay is the pause between NFeDistribuicaoDFe requests. It is a
// var so tests can drop it.
var nfeRequestDelay = 2 * time.Second

type nfeFetcher interface {
	DistNSU(ctx context.Context, cnpj string, cUFAutor int, ultNSU int64) (sefaz.DistResult, error)
}

// nfeSource walks the NF-e distribution of the Ambiente Nacional
// (NFeDistribuicaoDFe). It does not apply the company start policy: the
// SEFAZ queue already holds only the last 90 days.
type nfeSource struct {
	client   nfeFetcher
	repo     *store.NFeRepository
	xml      files.XMLStore
	log      *slog.Logger
	cUFAutor int
}

// NewNFeSource returns the Source for the NF-e distribution. cUFAutor is the
// IBGE code of the company's UF.
func NewNFeSource(client nfeFetcher, repo *store.NFeRepository, xml files.XMLStore, log *slog.Logger, cUFAutor int) Source {
	return &nfeSource{
		client:   client,
		repo:     repo,
		xml:      xml,
		log:      log,
		cUFAutor: cUFAutor,
	}
}

func (s *nfeSource) Kind() nfse.SyncSource {
	return nfse.SyncSourceNFe
}

func (s *nfeSource) Policy() SourcePolicy {
	return SourcePolicy{RequestDelay: nfeRequestDelay, RequestsPerHour: requestsPerHour(nfse.SyncSourceNFe)}
}

// Fetch asks for the documents after cursor; distBatch applies the stop
// rules.
func (s *nfeSource) Fetch(ctx context.Context, company *nfse.Company, cursor int64) (Batch, error) {
	resp, err := s.client.DistNSU(ctx, company.CNPJ, s.cUFAutor, cursor)
	if err != nil {
		return Batch{}, err
	}
	return distBatch(ctx, s.log, resp, cursor, isNFeEvent, "NF-e")
}

func isNFeEvent(schema string) bool {
	kind := nfe.ClassifySchema(schema)
	return kind == nfe.SchemaResEvento || kind == nfe.SchemaProcEventoNFe
}

func (s *nfeSource) ProcessItem(ctx context.Context, company *nfse.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	s.log.Log(ctx, logger.LevelTrace, "Processando documento NF-e", slog.Int64("nsu", item.NSU), slog.String("schema", item.Schema))

	payload, err := gzipxml.Decode(item.Payload, dfePayloadLimits)
	if err != nil {
		return ItemOutcome{}, &ProcessingError{Op: "decode document", NSU: item.NSU, Schema: item.Schema, Err: err}
	}

	tpAmb, err := sefaz.TpAmb(company.Environment)
	if err != nil {
		return ItemOutcome{}, err
	}

	switch nfe.ClassifySchema(item.Schema) {
	case nfe.SchemaResNFe:
		return s.processDocument(ctx, company, tpAmb, item, nfe.ParseResNFe, payload, commit)
	case nfe.SchemaProcNFe:
		return s.processDocument(ctx, company, tpAmb, item, nfe.ParseProcNFe, payload, commit)
	case nfe.SchemaResEvento:
		return s.processEvent(ctx, company, tpAmb, item, nfe.ParseResEvento, payload, commit)
	case nfe.SchemaProcEventoNFe:
		return s.processEvent(ctx, company, tpAmb, item, nfe.ParseProcEventoNFe, payload, commit)
	default:
		return s.processUnsupported(ctx, item, payload, commit)
	}
}

func (s *nfeSource) processDocument(ctx context.Context, company *nfse.Company, tpAmb string, item Item, parse func([]byte) (nfe.Document, error), payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	doc, err := parse(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse document", item, payload, err)
	}
	doc.TpAmb = checkTpAmb(doc.TpAmb, tpAmb, &doc.ParseWarnings)

	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	doc.RawHash = payload.SHA256

	params := store.ApplyNFeDocumentParams{
		Document:    doc,
		CompanyID:   company.ID,
		CompanyCNPJ: company.CNPJ,
		NSU:         item.NSU,
	}
	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		inserted, err := s.repo.ApplyDocumentTx(ctx, tx, params)
		return ItemOutcome{Inserted: inserted, Partial: doc.Completeness == nfe.CompletenessResumo}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply nfe document failed: %w", err)
	}
	return outcome, nil
}

// processEvent stores an event. An event for a chave the company does not
// see yet is skipped by policy, unless the company authored it (its own
// manifestação): that one is kept and linked when the document arrives.
func (s *nfeSource) processEvent(ctx context.Context, company *nfse.Company, tpAmb string, item Item, parse func([]byte) (nfe.Event, error), payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	ev, err := parse(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse event", item, payload, err)
	}
	ev.TpAmb = checkTpAmb(ev.TpAmb, tpAmb, &ev.ParseWarnings)

	authoredByCompany := cnpj.Clean(ev.AutorCNPJ) == company.CNPJ
	if !authoredByCompany {
		hasLocalDocument, err := s.repo.CompanyDocumentExists(ctx, company.ID, string(ev.ChaveAcesso))
		if err != nil {
			return ItemOutcome{}, fmt.Errorf("check local document for event failed: %w", err)
		}
		if !hasLocalDocument {
			outcome, err := commitSkip(ctx, commit, true)
			if err != nil {
				return ItemOutcome{}, err
			}
			s.log.InfoContext(ctx, "Evento NF-e descartado por não possuir documento local correspondente",
				slog.Int64("nsu", item.NSU),
				slog.String("tp_evento", ev.TpEvento))
			return outcome, nil
		}
	}

	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("event file save failed: %w", err)
	}
	ev.RawHash = payload.SHA256

	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		inserted, err := s.repo.ApplyEventTx(ctx, tx, store.ApplyNFeEventParams{Event: ev})
		return ItemOutcome{Inserted: inserted, IsEvent: true}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply nfe event failed: %w", err)
	}
	return outcome, nil
}

// processUnsupported keeps the XML of a schema nanci does not read and lets
// the cursor move past it.
func (s *nfeSource) processUnsupported(ctx context.Context, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	outcome, err := commit(ctx, func(*sql.Tx) (ItemOutcome, error) {
		return ItemOutcome{Unsupported: true}, nil
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("persist unsupported item progress failed: %w", err)
	}
	s.log.WarnContext(ctx, "Documento NF-e com schema não suportado ignorado",
		slog.Int64("nsu", item.NSU),
		slog.String("schema", item.Schema),
		slog.String("raw_hash", payload.SHA256))
	return outcome, nil
}

// parseError keeps the XML that failed to parse and wraps err so the loop
// can skip the item after repeated failures.
func (s *nfeSource) parseError(ctx context.Context, op string, item Item, payload gzipxml.Decoded, err error) error {
	return &ProcessingError{
		Op:         op,
		NSU:        item.NSU,
		Schema:     item.Schema,
		XMLPreview: xmlPreview(payload.XML),
		RawHash:    keepUnparsedXML(ctx, s.xml, s.log, payload),
		Err:        err,
	}
}
