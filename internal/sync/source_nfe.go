package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/syncstate"
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
	dfeItems
	client   nfeFetcher
	repo     *store.NFeRepository
	cUFAutor int
}

// NewNFeSource returns the Source for the NF-e distribution. cUFAutor is the
// IBGE code of the company's UF. Only Fetch uses client: a caller that only
// stores items through ProcessItem may pass a nil client and cUFAutor 0.
func NewNFeSource(client nfeFetcher, repo *store.NFeRepository, xml files.XMLStore, log *slog.Logger, cUFAutor int) Source {
	return &nfeSource{
		dfeItems: dfeItems{label: "NF-e", xml: xml, log: log, documentExists: repo.CompanyDocumentExists},
		client:   client,
		repo:     repo,
		cUFAutor: cUFAutor,
	}
}

func (s *nfeSource) Kind() syncstate.SyncSource {
	return syncstate.SyncSourceNFe
}

func (s *nfeSource) Policy() SourcePolicy {
	return SourcePolicy{RequestDelay: nfeRequestDelay, RequestsPerHour: requestsPerHour(syncstate.SyncSourceNFe)}
}

// Fetch asks for the documents after cursor; distBatch applies the stop
// rules.
func (s *nfeSource) Fetch(ctx context.Context, company *company.Company, cursor int64) (Batch, error) {
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

func (s *nfeSource) ProcessItem(ctx context.Context, company *company.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	payload, tpAmb, err := s.decode(ctx, company, item)
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
		return s.keepUnsupported(ctx, item, payload, commit)
	}
}

func (s *nfeSource) processDocument(ctx context.Context, company *company.Company, tpAmb string, item Item, parse func([]byte) (nfe.Document, error), payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	doc, err := parse(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse document", item, payload, err)
	}
	doc.TpAmb = CheckTpAmb(doc.TpAmb, tpAmb, &doc.ParseWarnings)

	partial := doc.Completeness == nfe.CompletenessResumo
	return s.keepDocument(ctx, payload, commit, partial, func(tx *sql.Tx, rawHash string) (bool, error) {
		doc.RawHash = rawHash
		return s.repo.ApplyDocumentTx(ctx, tx, store.ApplyNFeDocumentParams{
			Document:    doc,
			CompanyID:   company.ID,
			CompanyCNPJ: company.CNPJ,
			NSU:         &item.NSU,
		})
	})
}

// processEvent stores an event under the shared policy of keepEvent. The
// event the company authored is its own manifestação.
func (s *nfeSource) processEvent(ctx context.Context, company *company.Company, tpAmb string, item Item, parse func([]byte) (nfe.Event, error), payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	ev, err := parse(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse event", item, payload, err)
	}
	ev.TpAmb = CheckTpAmb(ev.TpAmb, tpAmb, &ev.ParseWarnings)

	keys := dfeEvent{chaveAcesso: string(ev.ChaveAcesso), autorCNPJ: ev.AutorCNPJ, tpEvento: ev.TpEvento}
	return s.keepEvent(ctx, company, item, payload, commit, keys, func(tx *sql.Tx, rawHash string) (bool, error) {
		ev.RawHash = rawHash
		return s.repo.ApplyEventTx(ctx, tx, store.ApplyNFeEventParams{Event: ev})
	})
}
