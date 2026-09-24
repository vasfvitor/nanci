package sync

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	dbstore "github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

// Access keys of the fixtures in internal/nfe/testdata.
const (
	nfeChaveProc      = "35260911222333000181550010000012341123456787" // resnfe.xml, procnfe.xml, proceventonfe-ciencia.xml
	nfeChaveCancelada = "35260911222333000181550010000012351234567894" // resnfe-cancelada.xml, resevento-cancelamento.xml
	nfeChaveDenegada  = "35260911222333000181550010000012361345678900" // procnfe-denegada.xml
	nfeCompanyCNPJ    = "70860312000150"                               // destinatário of the fixtures
)

// scriptedFetcher answers DistNSU and DistCTeNSU from a script keyed by the
// requested ultNSU. A cursor without a script gets cStat 137.
type scriptedFetcher struct {
	responses map[int64]sefaz.DistResult
	cursors   []int64
	cUFAutors []int
	cnpjs     []string
	services  []string // "nfe" or "cte" for each request
}

func (f *scriptedFetcher) DistNSU(_ context.Context, cnpj string, cUFAutor int, ultNSU int64) (sefaz.DistResult, error) {
	return f.answer("nfe", cnpj, cUFAutor, ultNSU)
}

func (f *scriptedFetcher) DistCTeNSU(_ context.Context, cnpj string, cUFAutor int, ultNSU int64) (sefaz.DistResult, error) {
	return f.answer("cte", cnpj, cUFAutor, ultNSU)
}

func (f *scriptedFetcher) answer(service, cnpj string, cUFAutor int, ultNSU int64) (sefaz.DistResult, error) {
	f.services = append(f.services, service)
	f.cursors = append(f.cursors, ultNSU)
	f.cUFAutors = append(f.cUFAutors, cUFAutor)
	f.cnpjs = append(f.cnpjs, cnpj)
	if resp, ok := f.responses[ultNSU]; ok {
		return resp, nil
	}
	return sefaz.DistResult{CStat: sefaz.CStatNenhumDocumento, UltNSU: ultNSU, MaxNSU: ultNSU}, nil
}

type nfeTestHelper struct {
	*testHelper
	repo *dbstore.NFeRepository
	xml  *mockXMLStore
}

// newNFeTestHelper returns a helper whose company is the destinatário of the
// NF-e fixtures, with UF SP.
func newNFeTestHelper(t *testing.T) *nfeTestHelper {
	t.Helper()
	db := storetest.OpenTestDB(t)

	cred := storetest.TestCredential("cred-1")
	if err := credential.NewStore(db).CreateCredential(context.Background(), cred); err != nil {
		t.Fatal(err)
	}
	company := storetest.TestCompany("comp-1", nfeCompanyCNPJ, nfse.EnvironmentProduction, cred)
	company.UF = "SP"
	company.SyncStartPolicy = nfse.SyncStartPolicyFromNow
	company.SyncStartDate = new(time.Now().UTC())
	if err := dbstore.NewCompanyRepository(db).CreateCompany(context.Background(), company); err != nil {
		t.Fatal(err)
	}

	originalDelay := nfeRequestDelay
	nfeRequestDelay = 0
	t.Cleanup(func() { nfeRequestDelay = originalDelay })

	return &nfeTestHelper{
		testHelper: &testHelper{t: t, db: db, store: NewStore(db), company: company, credential: cred},
		repo:       dbstore.NewNFeRepository(db),
		xml:        &mockXMLStore{},
	}
}

func (h *nfeTestHelper) source(fetcher nfeFetcher) Source {
	return NewNFeSource(fetcher, h.repo, h.xml, discardLogger(), 35)
}

// run syncs the NF-e source and returns the last progress event.
func (h *nfeTestHelper) run(fetcher nfeFetcher) (nfse.ProgressEvent, error) {
	h.t.Helper()
	var last nfse.ProgressEvent
	svc := NewSyncService(h.store, h.source(fetcher), discardLogger())
	err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, func(e nfse.ProgressEvent) {
		last = e
	})
	return last, err
}

func (h *nfeTestHelper) setNFeCursor(nsu int64) {
	h.t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := h.db.ExecContext(context.Background(), `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		VALUES (?, 'nfe', ?, ?, ?, ?, ?)
	`, string(h.company.ID), string(h.company.Environment), h.company.CNPJ, nsu, now, now)
	if err != nil {
		h.t.Fatalf("set nfe cursor: %v", err)
	}
}

func (h *nfeTestHelper) documents() map[string]nfe.CompanyDocument {
	h.t.Helper()
	docs, err := h.repo.ListCompanyDocuments(context.Background(), h.company.ID, nfe.DocumentFilter{})
	if err != nil {
		h.t.Fatal(err)
	}
	byChave := make(map[string]nfe.CompanyDocument, len(docs))
	for _, d := range docs {
		byChave[string(d.ChaveAcesso)] = d
	}
	return byChave
}

func (h *nfeTestHelper) events(chave string) []nfe.Event {
	h.t.Helper()
	events, err := h.repo.ListEventsByChave(context.Background(), chave)
	if err != nil {
		h.t.Fatal(err)
	}
	return events
}

// docZip builds a docZip from a fixture of internal/nfe/testdata.
func docZip(t *testing.T, nsu int64, schema, fixture string) sefaz.DocZip {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", fixture)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatal(err)
	}
	return sefaz.DocZip{NSU: nsu, Schema: schema, Content: mustEncodeGzipBase64(t, string(data))}
}

func assertRecentWait(t *testing.T, state SourceState, wantReason nfse.SyncStopReason) {
	t.Helper()
	if state.BlockedUntil == nil {
		t.Fatalf("blocked_until not set, want about an hour from now (%s)", wantReason)
	}
	if left := time.Until(*state.BlockedUntil); left < 55*time.Minute || left > 61*time.Minute {
		t.Errorf("blocked_until is %s away, want about an hour", left)
	}
	if state.BlockedReason != wantReason {
		t.Errorf("blocked_reason = %q, want %q", state.BlockedReason, wantReason)
	}
}

func TestNFeSourceStoresMixedBatchesAndStopsWhenCaughtUp(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 3, MaxNSU: 5, Docs: []sefaz.DocZip{
			docZip(t, 1, "resNFe_v1.01.xsd", "resnfe-cancelada.xml"),
			docZip(t, 2, "procNFe_v4.00.xsd", "procnfe.xml"),
			docZip(t, 3, "resEvento_v1.01.xsd", "resevento-cancelamento.xml"),
		}},
		3: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 5, MaxNSU: 5, Docs: []sefaz.DocZip{
			docZip(t, 5, "procNFe_v4.00.xsd", "procnfe-denegada.xml"),
		}},
	}}

	progress, err := h.run(fetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(fetcher.cursors) != 2 || fetcher.cursors[0] != 0 || fetcher.cursors[1] != 3 {
		t.Fatalf("fetch cursors = %v, want [0 3]", fetcher.cursors)
	}
	if fetcher.cUFAutors[0] != 35 || fetcher.cnpjs[0] != nfeCompanyCNPJ {
		t.Errorf("DistNSU got cUFAutor %d, CNPJ %s", fetcher.cUFAutors[0], fetcher.cnpjs[0])
	}
	cursor, maxNSU := h.sourceCursor(nfse.SyncSourceNFe)
	if cursor != 5 || !maxNSU.Valid || maxNSU.Int64 != 5 {
		t.Errorf("cursor = %d, max_nsu = %v, want 5 and 5", cursor, maxNSU)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)

	docs := h.documents()
	if len(docs) != 3 {
		t.Fatalf("documents = %d, want 3 (the start policy does not apply to NF-e)", len(docs))
	}
	if got := docs[nfeChaveCancelada]; got.Completeness != nfe.CompletenessResumo || got.Situacao != nfe.SituacaoCancelada {
		t.Errorf("cancelada = %s/%s, want resumo/cancelada", got.Completeness, got.Situacao)
	}
	if got := docs[nfeChaveProc]; got.Completeness != nfe.CompletenessCompleta || got.CompanyRole != nfe.CompanyRoleDestinatario {
		t.Errorf("proc = %s/%s, want completa/destinatario", got.Completeness, got.CompanyRole)
	}
	if got := docs[nfeChaveDenegada].Situacao; got != nfe.SituacaoDenegada {
		t.Errorf("denegada situacao = %s", got)
	}
	if got := len(h.events(nfeChaveCancelada)); got != 1 {
		t.Errorf("events of the cancelada = %d, want 1", got)
	}
	if progress.CompletasSaved != 2 || progress.ResumosSaved != 1 || progress.EventsSaved != 1 {
		t.Errorf("progress completas/resumos/events = %d/%d/%d, want 2/1/1", progress.CompletasSaved, progress.ResumosSaved, progress.EventsSaved)
	}
	if len(h.xml.stored) != 4 {
		t.Errorf("stored blobs = %d, want 4", len(h.xml.stored))
	}

	state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if state.InitialSyncDoneAt == nil {
		t.Error("nfe initial sync not marked after caught_up")
	}
	assertRecentWait(t, state, nfse.SyncStopReasonCaughtUp)

	// The NFS-e state of the company is untouched.
	h.assertInitialSyncCompleted(false)
	if got := h.countRows(`SELECT COUNT(*) FROM sync_state WHERE source = 'nfse'`); got != 0 {
		t.Errorf("nfse sync_state rows = %d, want 0", got)
	}
	nfseState, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFSe)
	if err != nil {
		t.Fatal(err)
	}
	if nfseState.BlockedUntil != nil {
		t.Errorf("nfse blocked_until = %v, want nil", nfseState.BlockedUntil)
	}
}

func TestNFeSourceUpgradesResumoToCompleta(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 2, Docs: []sefaz.DocZip{
			docZip(t, 1, "resNFe_v1.01.xsd", "resnfe.xml"),
		}},
		1: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 2, MaxNSU: 2, Docs: []sefaz.DocZip{
			docZip(t, 2, "procNFe_v4.00.xsd", "procnfe.xml"),
		}},
	}}

	progress, err := h.run(fetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	docs := h.documents()
	if len(docs) != 1 {
		t.Fatalf("documents = %d, want 1", len(docs))
	}
	got := docs[nfeChaveProc]
	if got.Completeness != nfe.CompletenessCompleta {
		t.Errorf("completeness = %s, want completa", got.Completeness)
	}
	if got.ResumoRawHash == "" || got.ResumoRawHash == got.RawHash {
		t.Errorf("resumo hash = %q, raw hash = %q: want the resumo blob kept apart", got.ResumoRawHash, got.RawHash)
	}
	if progress.ResumosSaved != 1 || progress.CompletasSaved != 1 {
		t.Errorf("progress resumos/completas = %d/%d, want 1/1", progress.ResumosSaved, progress.CompletasSaved)
	}
}

// TestNFeSourceRecordsTpAmb pulls in produção: a resumo and a resEvento,
// which carry no tpAmb, take the pull's; a procNFe of homologação keeps its
// own with a warning.
func TestNFeSourceRecordsTpAmb(t *testing.T) {
	h := newNFeTestHelper(t)
	denegada, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", "procnfe-denegada.xml"))
	if err != nil {
		t.Fatal(err)
	}
	homologacao := strings.ReplaceAll(string(denegada), "<tpAmb>1</tpAmb>", "<tpAmb>2</tpAmb>")
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 3, MaxNSU: 3, Docs: []sefaz.DocZip{
			docZip(t, 1, "resNFe_v1.01.xsd", "resnfe-cancelada.xml"),
			{NSU: 2, Schema: "procNFe_v4.00.xsd", Content: mustEncodeGzipBase64(t, homologacao)},
			docZip(t, 3, "resEvento_v1.01.xsd", "resevento-cancelamento.xml"),
		}},
	}}

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	docs := h.documents()
	if got := docs[nfeChaveCancelada]; got.TpAmb != "1" || len(got.ParseWarnings) != 0 {
		t.Errorf("resumo tpAmb = %q, warnings %v; want the pull's 1 and no warning", got.TpAmb, got.ParseWarnings)
	}
	got := docs[nfeChaveDenegada]
	if got.TpAmb != "2" {
		t.Errorf("homologação procNFe tpAmb = %q, want the XML's 2", got.TpAmb)
	}
	if len(got.ParseWarnings) != 1 || !strings.Contains(got.ParseWarnings[0], "tpAmb 2 differs from the queried tpAmb 1") {
		t.Errorf("homologação procNFe warnings = %v, want the tpAmb mismatch", got.ParseWarnings)
	}
	if events := h.events(nfeChaveCancelada); len(events) != 1 || events[0].TpAmb != "1" {
		t.Errorf("resEvento = %+v, want one with the pull's tpAmb 1", events)
	}
}

func TestNFeSourceKeepsOnlyOwnEventsWithoutLocalDocument(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 2, MaxNSU: 2, Docs: []sefaz.DocZip{
			docZip(t, 1, "resEvento_v1.01.xsd", "resevento-cancelamento.xml"),
			docZip(t, 2, "procEventoNFe_v1.00.xsd", "proceventonfe-ciencia.xml"),
		}},
	}}

	progress, err := h.run(fetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if got := len(h.events(nfeChaveCancelada)); got != 0 {
		t.Errorf("events of an unknown chave by another author = %d, want 0", got)
	}
	if got := len(h.events(nfeChaveProc)); got != 1 {
		t.Errorf("own ciência without the document = %d events, want 1", got)
	}
	if progress.EventsSkippedByPolicy != 1 || progress.EventsSaved != 1 {
		t.Errorf("events skipped/saved = %d/%d, want 1/1", progress.EventsSkippedByPolicy, progress.EventsSaved)
	}
	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 2 {
		t.Errorf("cursor = %d, want 2", cursor)
	}
}

func TestNFeSourceSkipsUnknownSchemaAndAdvances(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 1, Docs: []sefaz.DocZip{
			{NSU: 1, Schema: "resCTe_v1.00.xsd", Content: mustEncodeGzipBase64(t, "<resCTe/>")},
		}},
	}}

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 1 {
		t.Errorf("cursor = %d, want 1", cursor)
	}
	if got := len(h.documents()); got != 0 {
		t.Errorf("documents = %d, want 0", got)
	}
	if len(h.xml.stored) != 1 {
		t.Errorf("stored blobs = %d, want the unknown payload kept", len(h.xml.stored))
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
}

// A document that fails to parse is reported with an XML preview; the
// identifiers in it must not reach the error or the log in clear.
func TestNFeSourceParseFailureRedactsXMLPreview(t *testing.T) {
	h := newNFeTestHelper(t)
	const (
		emitenteCNPJ = "11222333000181"
		emitenteName = "DISTRIBUIDORA FICTICIA"
	)
	broken := `<resNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.01"><chNFe>` + nfeChaveProc + `</chNFe>` +
		`<CNPJ>` + emitenteCNPJ + `</CNPJ><xNome>` + emitenteName + `</xNome><IE>111222333444</IE><vNF>10.00`
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 1, Docs: []sefaz.DocZip{
			{NSU: 1, Schema: "resNFe_v1.01.xsd", Content: mustEncodeGzipBase64(t, broken)},
		}},
	}}

	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	svc := NewSyncService(h.store, NewNFeSource(fetcher, h.repo, h.xml, log, 35), log)
	// The item is skipped, and the failure logged, on the last attempt.
	for attempt := 1; attempt <= maxItemAttempts; attempt++ {
		err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
		if attempt < maxItemAttempts {
			var parseErr *ProcessingError
			if !errors.As(err, &parseErr) {
				t.Fatalf("attempt %d: error = %v, want a ProcessingError", attempt, err)
			}
			if !strings.Contains(err.Error(), "xml_preview=") {
				t.Fatalf("attempt %d: error lacks the XML preview: %v", attempt, err)
			}
			for _, clear := range []string{emitenteCNPJ, emitenteName, nfeChaveProc} {
				if strings.Contains(err.Error(), clear) {
					t.Errorf("attempt %d: error leaks %q: %v", attempt, clear, err)
				}
			}
			continue
		}
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
	}

	got := logs.String()
	if !strings.Contains(got, `"xml_preview":`) || !strings.Contains(got, "<CNPJ>11**********81</CNPJ>") {
		t.Fatalf("log lacks the masked XML preview:\n%s", got)
	}
	for _, clear := range []string{emitenteCNPJ, emitenteName, nfeChaveProc} {
		if strings.Contains(got, clear) {
			t.Errorf("log leaks %q:\n%s", clear, got)
		}
	}
}

func TestNFeSourceNenhumDocumentoNeverMovesCursorBack(t *testing.T) {
	h := newNFeTestHelper(t)
	h.setNFeCursor(10)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		10: {CStat: sefaz.CStatNenhumDocumento, UltNSU: 8, MaxNSU: 8},
	}}

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 10 {
		t.Errorf("cursor = %d, want 10", cursor)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
	state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	assertRecentWait(t, state, nfse.SyncStopReasonCaughtUp)
}

func TestNFeSourceConsumoIndevidoKeepsCursorAndBlocks(t *testing.T) {
	h := newNFeTestHelper(t)
	h.setNFeCursor(10)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		10: {CStat: sefaz.CStatConsumoIndevido, UltNSU: 4},
	}}

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 10 {
		t.Errorf("cursor = %d, want 10 (a lower ultNSU from 656 is ignored)", cursor)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonConsumoIndevido)
	state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	assertRecentWait(t, state, nfse.SyncStopReasonConsumoIndevido)
	if state.InitialSyncDoneAt != nil {
		t.Error("656 must not mark the initial sync")
	}
}

func TestNFeSourceRejectionFailsRunAsFetchError(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: 593, XMotivo: "CNPJ-Base consultado difere do CNPJ-Base do Certificado Digital"},
	}}

	_, err := h.run(fetcher)
	var rejection *sefaz.RejectionError
	if !errors.As(err, &rejection) || rejection.CStat != 593 {
		t.Fatalf("Sync error = %v, want the 593 rejection", err)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusFailed, nfse.SyncStopReasonFetchError)
}

// newNFePullTestManager is newPullTestManager with the company given a UF,
// an NF-e repository and a scripted SEFAZ client.
func newNFePullTestManager(t *testing.T, passwords CredentialProvider, fetcher *scriptedFetcher) (*Manager, *nfse.Company) {
	t.Helper()
	mgr, comp := newPullTestManager(t, passwords)
	if _, err := mgr.SyncRepo.db.ExecContext(context.Background(), `UPDATE companies SET uf = 'SP' WHERE id = ?`, string(comp.ID)); err != nil {
		t.Fatal(err)
	}
	mgr.NFeRepo = dbstore.NewNFeRepository(mgr.SyncRepo.db)

	originalNewSEFAZClient := newSEFAZClient
	originalDelay := nfeRequestDelay
	t.Cleanup(func() {
		newSEFAZClient = originalNewSEFAZClient
		nfeRequestDelay = originalDelay
	})
	nfeRequestDelay = 0
	newSEFAZClient = func(cfg sefaz.ClientConfig) (sefazFetcher, error) {
		if cfg.Environment != nfse.EnvironmentProduction || cfg.Certificate == nil {
			t.Errorf("SEFAZ client config = %+v", cfg)
		}
		return fetcher, nil
	}
	return mgr, comp
}

func TestPullNFeReportsLimitsAndBlocksAfterConsumoIndevido(t *testing.T) {
	passwords := &countingProvider{}
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 3, Docs: []sefaz.DocZip{
			docZip(t, 1, "resNFe_v1.01.xsd", "resnfe.xml"),
		}},
		1: {CStat: sefaz.CStatConsumoIndevido, UltNSU: 1},
	}}
	mgr, comp := newNFePullTestManager(t, passwords, fetcher)

	result, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceNFe})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if result.Source != nfse.SyncSourceNFe || result.Status != string(nfse.SyncStatusCompleted) || result.StopReason != string(nfse.SyncStopReasonConsumoIndevido) {
		t.Errorf("result source/status/reason = %s/%s/%s", result.Source, result.Status, result.StopReason)
	}
	if result.LastProcessedNSU != 1 || result.MaxNSU == nil || *result.MaxNSU != 3 {
		t.Errorf("cursor/max = %d/%v, want 1/3", result.LastProcessedNSU, result.MaxNSU)
	}
	if result.ResumosSaved != 1 || result.CompletasSaved != 0 {
		t.Errorf("resumos/completas = %d/%d, want 1/0", result.ResumosSaved, result.CompletasSaved)
	}
	if result.RequestsLastHour != 2 || result.RequestBudget != 20 {
		t.Errorf("requests = %d of %d, want 2 of 20", result.RequestsLastHour, result.RequestBudget)
	}
	if result.NextAllowedAt == nil || time.Until(*result.NextAllowedAt) < 55*time.Minute {
		t.Errorf("NextAllowedAt = %v, want about an hour from now", result.NextAllowedAt)
	}

	_, err = mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceNFe})
	var blocked *BlockedError
	if !errors.As(err, &blocked) || blocked.Reason != nfse.SyncStopReasonConsumoIndevido {
		t.Fatalf("second Pull error = %v, want *BlockedError for consumo_indevido", err)
	}
	if got := passwords.callCount(); got != 1 {
		t.Errorf("password prompts = %d, want 1 (the blocked pull must not prompt)", got)
	}
	if len(fetcher.cursors) != 2 {
		t.Errorf("SEFAZ requests = %d, want 2", len(fetcher.cursors))
	}
}

func TestPullNFeRequiresCompanyUFBeforePasswordPrompt(t *testing.T) {
	passwords := &countingProvider{}
	mgr, comp := newNFePullTestManager(t, passwords, &scriptedFetcher{})
	if _, err := mgr.SyncRepo.db.ExecContext(context.Background(), `UPDATE companies SET uf = '' WHERE id = ?`, string(comp.ID)); err != nil {
		t.Fatal(err)
	}

	_, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceNFe})
	if err == nil || !strings.Contains(err.Error(), "UF") {
		t.Fatalf("Pull error = %v, want the missing UF", err)
	}
	if got := passwords.callCount(); got != 0 {
		t.Errorf("password prompts = %d, want 0", got)
	}
}
