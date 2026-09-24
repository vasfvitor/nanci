package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
)

// cteRequestDelay is the pause between CTeDistribuicaoDFe requests. It is a
// var so tests can drop it.
var cteRequestDelay = 2 * time.Second

type cteFetcher interface {
	DistCTeNSU(ctx context.Context, cnpj string, cUFAutor int, ultNSU int64) (sefaz.DistResult, error)
}

// cteSource walks the CT-e distribution of the Ambiente Nacional
// (CTeDistribuicaoDFe). Like the NF-e source, it does not apply the company
// start policy: the queue already holds only the last 3 months.
type cteSource struct {
	client   cteFetcher
	repo     *store.CTeRepository
	xml      files.XMLStore
	log      *slog.Logger
	cUFAutor int
}

// NewCTeSource returns the Source for the CT-e distribution. cUFAutor is the
// IBGE code of the company's UF.
func NewCTeSource(client cteFetcher, repo *store.CTeRepository, xml files.XMLStore, log *slog.Logger, cUFAutor int) Source {
	return &cteSource{
		client:   client,
		repo:     repo,
		xml:      xml,
		log:      log,
		cUFAutor: cUFAutor,
	}
}

func (s *cteSource) Kind() nfse.SyncSource {
	return nfse.SyncSourceCTe
}

func (s *cteSource) Policy() SourcePolicy {
	return SourcePolicy{RequestDelay: cteRequestDelay, RequestsPerHour: requestsPerHour(nfse.SyncSourceCTe)}
}

// Fetch asks for the documents after cursor; distBatch applies the stop
// rules, the same as for the NF-e distribution.
func (s *cteSource) Fetch(ctx context.Context, company *nfse.Company, cursor int64) (Batch, error) {
	resp, err := s.client.DistCTeNSU(ctx, company.CNPJ, s.cUFAutor, cursor)
	if err != nil {
		return Batch{}, err
	}
	return distBatch(ctx, s.log, resp, cursor, isCTeEvent, "CT-e")
}

func isCTeEvent(schema string) bool {
	return cte.ClassifySchema(schema) == cte.SchemaProcEventoCTe
}

func (s *cteSource) ProcessItem(ctx context.Context, company *nfse.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	s.log.Log(ctx, logger.LevelTrace, "Processando documento CT-e", slog.Int64("nsu", item.NSU), slog.String("schema", item.Schema))

	payload, err := gzipxml.Decode(item.Payload, dfePayloadLimits)
	if err != nil {
		return ItemOutcome{}, &ProcessingError{Op: "decode document", NSU: item.NSU, Schema: item.Schema, Err: err}
	}

	tpAmb, err := sefaz.TpAmb(company.Environment)
	if err != nil {
		return ItemOutcome{}, err
	}

	switch cte.ClassifySchema(item.Schema) {
	case cte.SchemaProcCTe, cte.SchemaProcCTeOS, cte.SchemaProcGTVe, cte.SchemaProcCTeSimp:
		return s.processDocument(ctx, company, tpAmb, item, payload, commit)
	case cte.SchemaProcEventoCTe:
		return s.processEvent(ctx, company, tpAmb, item, payload, commit)
	default:
		return s.processUnsupported(ctx, item, payload, commit)
	}
}

func (s *cteSource) processDocument(ctx context.Context, company *nfse.Company, tpAmb string, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	doc, err := cte.ParseProcCTe(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse document", item, payload, err)
	}
	doc.TpAmb = checkTpAmb(doc.TpAmb, tpAmb, &doc.ParseWarnings)

	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	doc.RawHash = payload.SHA256

	params := store.ApplyCTeDocumentParams{
		Document:    doc,
		CompanyID:   company.ID,
		CompanyCNPJ: company.CNPJ,
		NSU:         item.NSU,
	}
	outcome, err := commit(ctx, func(tx *sql.Tx) (ItemOutcome, error) {
		inserted, err := s.repo.ApplyDocumentTx(ctx, tx, params)
		return ItemOutcome{Inserted: inserted}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply cte document failed: %w", err)
	}
	return outcome, nil
}

// processEvent stores an event. An event for a chave the company does not
// see yet is skipped by policy, unless the company authored it: that one is
// kept and linked when the document arrives. In practice this drops the
// MDF-e events the emitente receives about its own CT-e, which the
// distribution never delivers to it.
func (s *cteSource) processEvent(ctx context.Context, company *nfse.Company, tpAmb string, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	ev, err := cte.ParseProcEventoCTe(payload.XML)
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
			s.log.InfoContext(ctx, "Evento CT-e descartado por não possuir documento local correspondente",
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
		inserted, err := s.repo.ApplyEventTx(ctx, tx, store.ApplyCTeEventParams{Event: ev})
		return ItemOutcome{Inserted: inserted, IsEvent: true}, err
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("db apply cte event failed: %w", err)
	}
	return outcome, nil
}

// processUnsupported keeps the XML of a schema nanci does not read and lets
// the cursor move past it.
func (s *cteSource) processUnsupported(ctx context.Context, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	if err := s.xml.Store(payload.SHA256, payload.XML); err != nil {
		return ItemOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	outcome, err := commit(ctx, func(*sql.Tx) (ItemOutcome, error) {
		return ItemOutcome{Unsupported: true}, nil
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("persist unsupported item progress failed: %w", err)
	}
	s.log.WarnContext(ctx, "Documento CT-e com schema não suportado ignorado",
		slog.Int64("nsu", item.NSU),
		slog.String("schema", item.Schema),
		slog.String("raw_hash", payload.SHA256))
	return outcome, nil
}

// parseError keeps the XML that failed to parse and wraps err so the loop
// can skip the item after repeated failures.
func (s *cteSource) parseError(ctx context.Context, op string, item Item, payload gzipxml.Decoded, err error) error {
	return &ProcessingError{
		Op:         op,
		NSU:        item.NSU,
		Schema:     item.Schema,
		XMLPreview: xmlPreview(payload.XML),
		RawHash:    keepUnparsedXML(ctx, s.xml, s.log, payload),
		Err:        err,
	}
}
