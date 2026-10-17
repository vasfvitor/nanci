package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/syncstate"
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
	dfeItems
	client   cteFetcher
	repo     *store.CTeRepository
	cUFAutor int
}

// NewCTeSource returns the Source for the CT-e distribution. cUFAutor is the
// IBGE code of the company's UF. Only Fetch uses client: a caller that only
// stores items through ProcessItem may pass a nil client and cUFAutor 0.
func NewCTeSource(client cteFetcher, repo *store.CTeRepository, xml files.XMLStore, log *slog.Logger, cUFAutor int) Source {
	return &cteSource{
		dfeItems: dfeItems{label: "CT-e", xml: xml, log: log, documentExists: repo.CompanyDocumentExists},
		client:   client,
		repo:     repo,
		cUFAutor: cUFAutor,
	}
}

func (s *cteSource) Kind() syncstate.SyncSource {
	return syncstate.SyncSourceCTe
}

func (s *cteSource) Policy() SourcePolicy {
	return SourcePolicy{RequestDelay: cteRequestDelay, RequestsPerHour: requestsPerHour(syncstate.SyncSourceCTe)}
}

// Fetch asks for the documents after cursor; distBatch applies the stop
// rules, the same as for the NF-e distribution.
func (s *cteSource) Fetch(ctx context.Context, company *company.Company, cursor int64) (Batch, error) {
	resp, err := s.client.DistCTeNSU(ctx, company.CNPJ, s.cUFAutor, cursor)
	if err != nil {
		return Batch{}, err
	}
	return distBatch(ctx, s.log, resp, cursor, isCTeEvent, "CT-e")
}

func isCTeEvent(schema string) bool {
	return cte.ClassifySchema(schema) == cte.SchemaProcEventoCTe
}

func (s *cteSource) ProcessItem(ctx context.Context, company *company.Company, src SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	payload, tpAmb, err := s.decode(ctx, company, item)
	if err != nil {
		return ItemOutcome{}, err
	}

	switch cte.ClassifySchema(item.Schema) {
	case cte.SchemaProcCTe, cte.SchemaProcCTeOS, cte.SchemaProcGTVe, cte.SchemaProcCTeSimp:
		return s.processDocument(ctx, company, tpAmb, item, payload, commit)
	case cte.SchemaProcEventoCTe:
		return s.processEvent(ctx, company, tpAmb, item, payload, commit)
	default:
		return s.keepUnsupported(ctx, item, payload, commit)
	}
}

func (s *cteSource) processDocument(ctx context.Context, company *company.Company, tpAmb string, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	doc, err := cte.ParseProcCTe(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse document", item, payload, err)
	}
	doc.TpAmb = CheckTpAmb(doc.TpAmb, tpAmb, &doc.ParseWarnings)

	return s.keepDocument(ctx, payload, commit, false, func(tx *sql.Tx, rawHash string) (bool, error) {
		doc.RawHash = rawHash
		return s.repo.ApplyDocumentTx(ctx, tx, store.ApplyCTeDocumentParams{
			Document:    doc,
			CompanyID:   company.ID,
			CompanyCNPJ: company.CNPJ,
			NSU:         &item.NSU,
		})
	})
}

// processEvent stores an event under the shared policy of keepEvent. In
// practice the policy drops the MDF-e events the emitente receives about its
// own CT-e, which the distribution never delivers to it.
func (s *cteSource) processEvent(ctx context.Context, company *company.Company, tpAmb string, item Item, payload gzipxml.Decoded, commit CommitFunc) (ItemOutcome, error) {
	ev, err := cte.ParseProcEventoCTe(payload.XML)
	if err != nil {
		return ItemOutcome{}, s.parseError(ctx, "parse event", item, payload, err)
	}
	ev.TpAmb = CheckTpAmb(ev.TpAmb, tpAmb, &ev.ParseWarnings)

	keys := dfeEvent{chaveAcesso: string(ev.ChaveAcesso), autorCNPJ: ev.AutorCNPJ, tpEvento: ev.TpEvento}
	return s.keepEvent(ctx, company, item, payload, commit, keys, func(tx *sql.Tx, rawHash string) (bool, error) {
		ev.RawHash = rawHash
		return s.repo.ApplyEventTx(ctx, tx, store.ApplyCTeEventParams{Event: ev})
	})
}
