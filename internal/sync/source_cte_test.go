package sync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	dbstore "github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

// Access keys of the fixtures in internal/cte/testdata. The mock company is a
// party of every fixture.
const (
	cteChaveProc  = "35260912345678000195570010000001011123456784" // procte.xml, proceventocte-cancelamento.xml
	cteChaveToma4 = "35260912345678000195570010000001021234567891" // procte-toma4.xml, proceventocte-comprovante.xml
	cteChaveV200  = "35260912345678000195570010000001031345678907" // procte-v200-toma03.xml, tpAmb 2
	cteChaveOS    = "35260912345678000195670010000001041456789014" // procteos.xml
	cteChaveGTVe  = "35260912345678000195640010000001051567890127" // procgtve.xml
	cteChaveSimp  = "35260912345678000195570020000001061678901234" // proctesimp.xml
	cteEmitente   = "12345678000195"                               // emitente and event author of the fixtures
)

// scriptedCTeFetcher answers DistCTeNSU from the same script format as
// scriptedFetcher.
type scriptedCTeFetcher struct {
	scriptedFetcher
}

func (f *scriptedCTeFetcher) DistCTeNSU(ctx context.Context, cnpj string, cUFAutor int, ultNSU int64) (sefaz.DistResult, error) {
	return f.DistNSU(ctx, cnpj, cUFAutor, ultNSU)
}

func newScriptedCTeFetcher(responses map[int64]sefaz.DistResult) *scriptedCTeFetcher {
	return &scriptedCTeFetcher{scriptedFetcher{responses: responses}}
}

type cteTestHelper struct {
	*testHelper
	repo *dbstore.CTeRepository
	xml  *mockXMLStore
}

// newCTeTestHelper returns a helper whose company is the mock CNPJ of the
// CT-e fixtures, in produção, with UF SP.
func newCTeTestHelper(t *testing.T) *cteTestHelper {
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

	originalDelay := cteRequestDelay
	cteRequestDelay = 0
	t.Cleanup(func() { cteRequestDelay = originalDelay })

	return &cteTestHelper{
		testHelper: &testHelper{t: t, db: db, store: NewStore(db), company: company, credential: cred},
		repo:       dbstore.NewCTeRepository(db),
		xml:        &mockXMLStore{},
	}
}

func (h *cteTestHelper) source(fetcher cteFetcher) Source {
	return NewCTeSource(fetcher, h.repo, h.xml, discardLogger(), 35)
}

// run syncs the CT-e source and returns the last progress event.
func (h *cteTestHelper) run(fetcher cteFetcher) (nfse.ProgressEvent, error) {
	h.t.Helper()
	var last nfse.ProgressEvent
	svc := NewSyncService(h.store, h.source(fetcher), discardLogger())
	err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, func(e nfse.ProgressEvent) {
		last = e
	})
	return last, err
}

func (h *cteTestHelper) setCTeCursor(nsu int64) {
	h.t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := h.db.ExecContext(context.Background(), `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		VALUES (?, 'cte', ?, ?, ?, ?, ?)
	`, string(h.company.ID), string(h.company.Environment), h.company.CNPJ, nsu, now, now)
	if err != nil {
		h.t.Fatalf("set cte cursor: %v", err)
	}
}

func (h *cteTestHelper) documents() map[string]cte.CompanyDocument {
	h.t.Helper()
	docs, err := h.repo.ListCompanyDocuments(context.Background(), h.company.ID, cte.DocumentFilter{})
	if err != nil {
		h.t.Fatal(err)
	}
	byChave := make(map[string]cte.CompanyDocument, len(docs))
	for _, d := range docs {
		byChave[string(d.ChaveAcesso)] = d
	}
	return byChave
}

func (h *cteTestHelper) events(chave string) []cte.Event {
	h.t.Helper()
	events, err := h.repo.ListEventsByChave(context.Background(), chave)
	if err != nil {
		h.t.Fatal(err)
	}
	return events
}

func readCTeFixture(t *testing.T, fixture string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "cte", "testdata", fixture)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// cteDocZip builds a docZip from a fixture of internal/cte/testdata.
func cteDocZip(t *testing.T, nsu int64, schema, fixture string) sefaz.DocZip {
	t.Helper()
	return sefaz.DocZip{NSU: nsu, Schema: schema, Content: mustEncodeGzipBase64(t, readCTeFixture(t, fixture))}
}

func TestCTeSourceStoresEveryDocumentKindAndStopsWhenCaughtUp(t *testing.T) {
	h := newCTeTestHelper(t)
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 3, MaxNSU: 6, Docs: []sefaz.DocZip{
			cteDocZip(t, 1, "procCTe_v4.00.xsd", "procte.xml"),
			cteDocZip(t, 2, "procCTeOS_v4.00.xsd", "procteos.xml"),
			cteDocZip(t, 3, "procEventoCTe_v4.00.xsd", "proceventocte-cancelamento.xml"),
		}},
		3: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 6, MaxNSU: 6, Docs: []sefaz.DocZip{
			cteDocZip(t, 4, "procGTVe_v4.00.xsd", "procgtve.xml"),
			cteDocZip(t, 5, "procCTeSimp_v4.00.xsd", "proctesimp.xml"),
			cteDocZip(t, 6, "procCTe_v4.00.xsd", "procte-toma4.xml"),
		}},
	})

	progress, err := h.run(fetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(fetcher.cursors) != 2 || fetcher.cursors[0] != 0 || fetcher.cursors[1] != 3 {
		t.Fatalf("fetch cursors = %v, want [0 3]", fetcher.cursors)
	}
	if fetcher.cUFAutors[0] != 35 || fetcher.cnpjs[0] != nfeCompanyCNPJ {
		t.Errorf("DistCTeNSU got cUFAutor %d, CNPJ %s", fetcher.cUFAutors[0], fetcher.cnpjs[0])
	}
	cursor, maxNSU := h.sourceCursor(nfse.SyncSourceCTe)
	if cursor != 6 || !maxNSU.Valid || maxNSU.Int64 != 6 {
		t.Errorf("cursor = %d, max_nsu = %v, want 6 and 6", cursor, maxNSU)
	}
	h.assertSourceRun(nfse.SyncSourceCTe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)

	docs := h.documents()
	if len(docs) != 5 {
		t.Fatalf("documents = %d, want 5", len(docs))
	}
	wantTipos := map[string]cte.TipoDocumento{
		cteChaveProc:  cte.TipoDocumentoCTe,
		cteChaveOS:    cte.TipoDocumentoCTeOS,
		cteChaveGTVe:  cte.TipoDocumentoGTVe,
		cteChaveSimp:  cte.TipoDocumentoCTeSimplificado,
		cteChaveToma4: cte.TipoDocumentoCTe,
	}
	for chave, want := range wantTipos {
		got, ok := docs[chave]
		if !ok {
			t.Errorf("document %s missing", chave)
			continue
		}
		if got.TipoDocumento != want || got.TpAmb != "1" {
			t.Errorf("document %s = %s/tpAmb %s, want %s/1", chave, got.TipoDocumento, got.TpAmb, want)
		}
	}
	if got := docs[cteChaveProc]; got.Situacao != cte.SituacaoCancelada || got.CompanyRole != cte.CompanyRoleTomador {
		t.Errorf("procte = %s/%s, want cancelada/tomador", got.Situacao, got.CompanyRole)
	}
	if got := len(h.events(cteChaveProc)); got != 1 {
		t.Errorf("events of the cancelada = %d, want 1", got)
	}
	if progress.CompletasSaved != 5 || progress.ResumosSaved != 0 || progress.EventsSaved != 1 {
		t.Errorf("progress completas/resumos/events = %d/%d/%d, want 5/0/1", progress.CompletasSaved, progress.ResumosSaved, progress.EventsSaved)
	}
	if len(h.xml.stored) != 6 {
		t.Errorf("stored blobs = %d, want 6", len(h.xml.stored))
	}

	state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceCTe)
	if err != nil {
		t.Fatal(err)
	}
	if state.InitialSyncDoneAt == nil {
		t.Error("cte initial sync not marked after caught_up")
	}
	assertRecentWait(t, state, nfse.SyncStopReasonCaughtUp)

	// The NF-e state of the company is untouched.
	nfeState, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if nfeState.BlockedUntil != nil || nfeState.InitialSyncDoneAt != nil {
		t.Errorf("nfe state = %+v, want untouched", nfeState)
	}
}

func TestCTeSourceWarnsOnTpAmbMismatch(t *testing.T) {
	h := newCTeTestHelper(t)
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 1, Docs: []sefaz.DocZip{
			cteDocZip(t, 1, "procCTe_v2.00.xsd", "procte-v200-toma03.xml"),
		}},
	})

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	got, ok := h.documents()[cteChaveV200]
	if !ok {
		t.Fatal("homologação CT-e not stored")
	}
	if got.TpAmb != "2" {
		t.Errorf("tpAmb = %q, want the XML's 2", got.TpAmb)
	}
	var mismatch bool
	for _, w := range got.ParseWarnings {
		if strings.Contains(w, "tpAmb 2 differs from the queried tpAmb 1") {
			mismatch = true
		}
	}
	if !mismatch {
		t.Errorf("warnings = %v, want the tpAmb mismatch", got.ParseWarnings)
	}
}

func TestCTeSourceKeepsOnlyOwnEventsWithoutLocalDocument(t *testing.T) {
	h := newCTeTestHelper(t)
	ownEvent := strings.Replace(readCTeFixture(t, "proceventocte-cancelamento.xml"),
		"<CNPJ>"+cteEmitente+"</CNPJ>", "<CNPJ>"+nfeCompanyCNPJ+"</CNPJ>", 1)
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 2, MaxNSU: 2, Docs: []sefaz.DocZip{
			cteDocZip(t, 1, "procEventoCTe_v4.00.xsd", "proceventocte-comprovante.xml"),
			{NSU: 2, Schema: "procEventoCTe_v4.00.xsd", Content: mustEncodeGzipBase64(t, ownEvent)},
		}},
	})

	progress, err := h.run(fetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if got := len(h.events(cteChaveToma4)); got != 0 {
		t.Errorf("events of an unknown chave by another author = %d, want 0", got)
	}
	if got := len(h.events(cteChaveProc)); got != 1 {
		t.Errorf("own event without the document = %d events, want 1", got)
	}
	if progress.EventsSkippedByPolicy != 1 || progress.EventsSaved != 1 {
		t.Errorf("events skipped/saved = %d/%d, want 1/1", progress.EventsSkippedByPolicy, progress.EventsSaved)
	}
	if got := len(h.documents()); got != 0 {
		t.Errorf("documents = %d, want 0", got)
	}
	if cursor, _ := h.sourceCursor(nfse.SyncSourceCTe); cursor != 2 {
		t.Errorf("cursor = %d, want 2", cursor)
	}
}

func TestCTeSourceSkipsUnknownSchemaAndAdvances(t *testing.T) {
	h := newCTeTestHelper(t)
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 1, Docs: []sefaz.DocZip{
			{NSU: 1, Schema: "procMDFe_v3.00.xsd", Content: mustEncodeGzipBase64(t, "<mdfeProc/>")},
		}},
	})

	progress, err := h.run(fetcher)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceCTe); cursor != 1 {
		t.Errorf("cursor = %d, want 1", cursor)
	}
	if got := len(h.documents()); got != 0 {
		t.Errorf("documents = %d, want 0", got)
	}
	if len(h.xml.stored) != 1 {
		t.Errorf("stored blobs = %d, want the unknown payload kept", len(h.xml.stored))
	}
	if progress.CompletasSaved != 0 {
		t.Errorf("completas = %d, want 0 for an unsupported item", progress.CompletasSaved)
	}
	h.assertSourceRun(nfse.SyncSourceCTe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
}

// A document that keeps failing to parse is retried on each pull, then kept
// as a blob and skipped so the cursor moves on.
func TestCTeSourceSkipsDocumentAfterRepeatedParseFailures(t *testing.T) {
	h := newCTeTestHelper(t)
	broken := `<cteProc xmlns="http://www.portalfiscal.inf.br/cte" versao="4.00"><CTe><infCte Id="CTe` + cteChaveProc + `">`
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 1, Docs: []sefaz.DocZip{
			{NSU: 1, Schema: "procCTe_v4.00.xsd", Content: mustEncodeGzipBase64(t, broken)},
		}},
	})

	for attempt := 1; attempt < maxItemAttempts; attempt++ {
		_, err := h.run(fetcher)
		var parseErr *ProcessingError
		if !errors.As(err, &parseErr) {
			t.Fatalf("attempt %d: error = %v, want a ProcessingError", attempt, err)
		}
		if cursor, _ := h.sourceCursor(nfse.SyncSourceCTe); cursor != 0 {
			t.Fatalf("attempt %d: cursor = %d, want 0 while retrying", attempt, cursor)
		}
	}
	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("attempt %d: %v", maxItemAttempts, err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceCTe); cursor != 1 {
		t.Errorf("cursor = %d, want 1 after the item is skipped", cursor)
	}
	if got := len(h.documents()); got != 0 {
		t.Errorf("documents = %d, want 0", got)
	}
	if len(h.xml.stored) != 1 {
		t.Errorf("stored blobs = %d, want the unparsed payload kept", len(h.xml.stored))
	}
	if got := len(fetcher.cursors); got != maxItemAttempts {
		t.Errorf("SEFAZ requests = %d, want %d (each attempt spends budget)", got, maxItemAttempts)
	}
}

func TestCTeSourceNenhumDocumentoNeverMovesCursorBack(t *testing.T) {
	h := newCTeTestHelper(t)
	h.setCTeCursor(10)
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		10: {CStat: sefaz.CStatNenhumDocumento, UltNSU: 8, MaxNSU: 8},
	})

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceCTe); cursor != 10 {
		t.Errorf("cursor = %d, want 10", cursor)
	}
	h.assertSourceRun(nfse.SyncSourceCTe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
	state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceCTe)
	if err != nil {
		t.Fatal(err)
	}
	assertRecentWait(t, state, nfse.SyncStopReasonCaughtUp)
}

func TestCTeSourceConsumoIndevidoKeepsCursorAndBlocks(t *testing.T) {
	tests := []struct {
		name       string
		ultNSU     int64
		wantCursor int64
	}{
		{"lower ultNSU", 4, 10},
		{"higher ultNSU", 12, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newCTeTestHelper(t)
			h.setCTeCursor(10)
			fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
				10: {CStat: sefaz.CStatConsumoIndevido, UltNSU: tt.ultNSU},
			})

			if _, err := h.run(fetcher); err != nil {
				t.Fatalf("Sync: %v", err)
			}

			if cursor, _ := h.sourceCursor(nfse.SyncSourceCTe); cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d (the cursor only moves forward)", cursor, tt.wantCursor)
			}
			h.assertSourceRun(nfse.SyncSourceCTe, nfse.SyncStatusCompleted, nfse.SyncStopReasonConsumoIndevido)
			state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceCTe)
			if err != nil {
				t.Fatal(err)
			}
			assertRecentWait(t, state, nfse.SyncStopReasonConsumoIndevido)
			if state.InitialSyncDoneAt != nil {
				t.Error("656 must not mark the initial sync")
			}
		})
	}
}

func TestCTeSourceRejectionFailsRunAsFetchError(t *testing.T) {
	h := newCTeTestHelper(t)
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: 593, XMotivo: "CNPJ-Base consultado difere do CNPJ-Base do Certificado Digital"},
	})

	_, err := h.run(fetcher)
	var rejection *sefaz.RejectionError
	if !errors.As(err, &rejection) || rejection.CStat != 593 {
		t.Fatalf("Sync error = %v, want the 593 rejection", err)
	}
	h.assertSourceRun(nfse.SyncSourceCTe, nfse.SyncStatusFailed, nfse.SyncStopReasonFetchError)
}

// newCTePullTestManager is newPullTestManager with the company given a UF, a
// CT-e repository and a scripted CT-e client. The NF-e client must not be
// built.
func newCTePullTestManager(t *testing.T, passwords CredentialProvider, fetcher cteFetcher) (*Manager, *nfse.Company) {
	t.Helper()
	mgr, comp := newPullTestManager(t, passwords)
	if _, err := mgr.SyncRepo.db.ExecContext(context.Background(), `UPDATE companies SET uf = 'SP' WHERE id = ?`, string(comp.ID)); err != nil {
		t.Fatal(err)
	}
	mgr.CTeRepo = dbstore.NewCTeRepository(mgr.SyncRepo.db)

	originalNewSEFAZClient := newSEFAZClient
	originalNewSEFAZCTeClient := newSEFAZCTeClient
	originalDelay := cteRequestDelay
	t.Cleanup(func() {
		newSEFAZClient = originalNewSEFAZClient
		newSEFAZCTeClient = originalNewSEFAZCTeClient
		cteRequestDelay = originalDelay
	})
	cteRequestDelay = 0
	newSEFAZClient = func(sefaz.ClientConfig) (nfeFetcher, error) {
		t.Error("a CT-e pull built the NF-e client")
		return nil, errors.New("unexpected NF-e client")
	}
	newSEFAZCTeClient = func(cfg sefaz.ClientConfig) (cteFetcher, error) {
		if cfg.Environment != nfse.EnvironmentProduction || cfg.Certificate == nil {
			t.Errorf("SEFAZ client config = %+v", cfg)
		}
		return fetcher, nil
	}
	return mgr, comp
}

func TestPullCTeStoresDocumentsReportsLimitsAndBlocks(t *testing.T) {
	passwords := &countingProvider{}
	fetcher := newScriptedCTeFetcher(map[int64]sefaz.DistResult{
		0: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 1, MaxNSU: 3, Docs: []sefaz.DocZip{
			cteDocZip(t, 1, "procCTe_v4.00.xsd", "procte.xml"),
		}},
		1: {CStat: sefaz.CStatConsumoIndevido, UltNSU: 1},
	})
	mgr, comp := newCTePullTestManager(t, passwords, fetcher)
	ctx := context.Background()

	result, err := mgr.Pull(ctx, PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceCTe})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if result.Source != nfse.SyncSourceCTe || result.Status != string(nfse.SyncStatusCompleted) || result.StopReason != string(nfse.SyncStopReasonConsumoIndevido) {
		t.Errorf("result source/status/reason = %s/%s/%s", result.Source, result.Status, result.StopReason)
	}
	if result.LastProcessedNSU != 1 || result.MaxNSU == nil || *result.MaxNSU != 3 {
		t.Errorf("cursor/max = %d/%v, want 1/3", result.LastProcessedNSU, result.MaxNSU)
	}
	if result.CompletasSaved != 1 || result.ResumosSaved != 0 {
		t.Errorf("completas/resumos = %d/%d, want 1/0", result.CompletasSaved, result.ResumosSaved)
	}
	if result.RequestsLastHour != 2 || result.RequestBudget != CTeRequestsPerHour {
		t.Errorf("requests = %d of %d, want 2 of %d", result.RequestsLastHour, result.RequestBudget, CTeRequestsPerHour)
	}
	if result.NextAllowedAt == nil || time.Until(*result.NextAllowedAt) < 55*time.Minute {
		t.Errorf("NextAllowedAt = %v, want about an hour from now", result.NextAllowedAt)
	}
	if purpose := passwords.lastPurpose(); purpose != "Sincronização CT-e" {
		t.Errorf("password purpose = %q, want Sincronização CT-e", purpose)
	}

	docs, err := mgr.CTeRepo.ListCompanyDocuments(ctx, comp.ID, cte.DocumentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || string(docs[0].ChaveAcesso) != cteChaveProc || docs[0].CompanyRole != cte.CompanyRoleRemetente {
		t.Fatalf("documents = %+v, want procte.xml with the company as remetente", docs)
	}

	_, err = mgr.Pull(ctx, PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceCTe})
	var blocked *BlockedError
	if !errors.As(err, &blocked) || blocked.Reason != nfse.SyncStopReasonConsumoIndevido || blocked.Source != nfse.SyncSourceCTe {
		t.Fatalf("second Pull error = %v, want *BlockedError for CT-e consumo_indevido", err)
	}
	if !strings.Contains(err.Error(), "CT-e") {
		t.Errorf("blocked error %q does not name CT-e", err)
	}
	if got := passwords.callCount(); got != 1 {
		t.Errorf("password prompts = %d, want 1 (the blocked pull must not prompt)", got)
	}
	if len(fetcher.cursors) != 2 {
		t.Errorf("SEFAZ requests = %d, want 2", len(fetcher.cursors))
	}

	// The NF-e budget and block are counted apart.
	nfeLimits, err := mgr.SourceLimits(ctx, comp.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if nfeLimits.RequestsLastHour != 0 || nfeLimits.NextAllowedAt != nil {
		t.Errorf("NF-e limits = %+v, want untouched by CT-e pulls", nfeLimits)
	}
}

func TestPullCTeRequiresCompanyUFAndRepoBeforePasswordPrompt(t *testing.T) {
	passwords := &countingProvider{}
	mgr, comp := newCTePullTestManager(t, passwords, newScriptedCTeFetcher(nil))
	if _, err := mgr.SyncRepo.db.ExecContext(context.Background(), `UPDATE companies SET uf = '' WHERE id = ?`, string(comp.ID)); err != nil {
		t.Fatal(err)
	}

	_, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceCTe})
	if err == nil || !strings.Contains(err.Error(), "UF") {
		t.Fatalf("Pull error = %v, want the missing UF", err)
	}

	mgr.CTeRepo = nil
	_, err = mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ, Source: nfse.SyncSourceCTe})
	if err == nil || !strings.Contains(err.Error(), "CT-e") {
		t.Fatalf("Pull error = %v, want the missing CT-e repository", err)
	}
	if got := passwords.callCount(); got != 0 {
		t.Errorf("password prompts = %d, want 0", got)
	}
}
