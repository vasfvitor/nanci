package syncservice

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

type fetchResult struct {
	response *adn.DocumentResponse
	err      error
}

type fakeFetcher struct {
	results  []fetchResult
	requests []adn.DistributionRequest
}

func (f *fakeFetcher) FetchDocuments(_ context.Context, req adn.DistributionRequest) (*adn.DocumentResponse, error) {
	f.requests = append(f.requests, req)
	if len(f.results) == 0 {
		return nil, fmt.Errorf("unexpected fetch for NSU %d", req.LastNSU)
	}

	result := f.results[0]
	f.results = f.results[1:]
	return result.response, result.err
}

func TestSyncAdvancesCheckpointToUltNSUOnEmptyBatch(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 10)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{UltNSU: 15, MaxNSU: 15, Docs: nil}},
		},
	}
	svc.apiClient = fetcher

	if err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil); err != nil {
		t.Fatalf("Sync first run: %v", err)
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany after first run: %v", err)
	}
	if reloaded.LastNSU != 15 {
		t.Fatalf("LastNSU after empty batch = %d, want 15", reloaded.LastNSU)
	}

	run := readLatestSyncRun(t, rawDB)
	if run.Status != "completed" || run.ToNSU != 15 {
		t.Fatalf("unexpected sync run after empty batch: %+v", run)
	}

	secondFetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{UltNSU: 15, MaxNSU: 15, Docs: nil}},
		},
	}
	svc.apiClient = secondFetcher
	if err := svc.Sync(context.Background(), reloaded, testCredential(), "exact_certificate_cnpj", nil); err != nil {
		t.Fatalf("Sync second run: %v", err)
	}
	if len(secondFetcher.requests) != 1 || secondFetcher.requests[0].LastNSU != 15 || secondFetcher.requests[0].ConsultationCNPJ != company.CNPJ {
		t.Fatalf("second fetch requests = %+v, want consultation on company cnpj at nsu 15", secondFetcher.requests)
	}
}

func TestSyncCommitsUltNSUEvenWhenLastEnvelopeIsLower(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 10)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{
				UltNSU: 25,
				MaxNSU: 25,
				Docs: []adn.DocumentEnvelope{
					makeDocumentEnvelope(11, "CHAVE-11", "11111111000111", "22222222000122"),
					makeDocumentEnvelope(12, "CHAVE-12", "11111111000111", "22222222000122"),
				},
			}},
		},
	}
	svc.apiClient = fetcher

	if err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if reloaded.LastNSU != 25 {
		t.Fatalf("LastNSU = %d, want 25", reloaded.LastNSU)
	}

	run := readLatestSyncRun(t, rawDB)
	if run.Status != "completed" || run.ToNSU != 25 || run.DocumentsFound != 2 {
		t.Fatalf("unexpected sync run: %+v", run)
	}
}

func TestSyncFailsWithoutAdvancingCheckpointWhenFirstItemFails(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 10)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{
				UltNSU: 20,
				MaxNSU: 20,
				Docs: []adn.DocumentEnvelope{
					{NSU: 11, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: "not-base64"},
				},
			}},
		},
	}
	svc.apiClient = fetcher

	err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil)
	if err == nil {
		t.Fatal("Sync error = nil, want failure")
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if reloaded.LastNSU != 10 {
		t.Fatalf("LastNSU after first-item failure = %d, want 10", reloaded.LastNSU)
	}

	run := readLatestSyncRun(t, rawDB)
	if run.Status != "failed" || run.ToNSU != 10 || run.ErrorsCount != 1 || run.DocumentsFound != 0 {
		t.Fatalf("unexpected sync run: %+v", run)
	}
}

func TestSyncFailsOnLaterItemAndRetriesFromLastCommittedNSU(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 9)

	firstFetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{
				UltNSU: 30,
				MaxNSU: 30,
				Docs: []adn.DocumentEnvelope{
					makeDocumentEnvelope(10, "CHAVE-10", company.CNPJ, "22222222000122"),
					{NSU: 11, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: "not-base64"},
				},
			}},
		},
	}
	svc.apiClient = firstFetcher

	err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil)
	if err == nil {
		t.Fatal("first Sync error = nil, want failure")
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany after failed run: %v", err)
	}
	if reloaded.LastNSU != 10 {
		t.Fatalf("LastNSU after later-item failure = %d, want 10", reloaded.LastNSU)
	}

	firstRun := readLatestSyncRun(t, rawDB)
	if firstRun.Status != "failed" || firstRun.ToNSU != 10 || firstRun.ErrorsCount != 1 || firstRun.DocumentsFound != 1 {
		t.Fatalf("unexpected first sync run: %+v", firstRun)
	}

	secondFetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{
				UltNSU: 30,
				MaxNSU: 30,
				Docs: []adn.DocumentEnvelope{
					makeDocumentEnvelope(11, "CHAVE-11", company.CNPJ, "22222222000122"),
				},
			}},
		},
	}
	svc.apiClient = secondFetcher
	if err := svc.Sync(context.Background(), reloaded, testCredential(), "exact_certificate_cnpj", nil); err != nil {
		t.Fatalf("second Sync: %v", err)
	}

	if len(secondFetcher.requests) != 1 || secondFetcher.requests[0].LastNSU != 10 || secondFetcher.requests[0].ConsultationCNPJ != company.CNPJ {
		t.Fatalf("second fetch requests = %+v, want consultation on company cnpj at nsu 10", secondFetcher.requests)
	}

	reloadedAgain, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany after retry: %v", err)
	}
	if reloadedAgain.LastNSU != 30 {
		t.Fatalf("LastNSU after retry = %d, want 30", reloadedAgain.LastNSU)
	}
}

func TestSyncDoesNotAdvanceCheckpointOnTransportError(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 12)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{err: fmt.Errorf("transport down")},
		},
	}
	svc.apiClient = fetcher

	err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil)
	if err == nil {
		t.Fatal("Sync error = nil, want failure")
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if reloaded.LastNSU != 12 {
		t.Fatalf("LastNSU after transport error = %d, want 12", reloaded.LastNSU)
	}

	run := readLatestSyncRun(t, rawDB)
	if run.Status != "failed" || run.ToNSU != 12 {
		t.Fatalf("unexpected sync run: %+v", run)
	}
}

func TestSyncMarksRunInterruptedOnContextCancellation(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 12)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{err: context.Canceled},
		},
	}
	svc.apiClient = fetcher

	err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil)
	if err == nil {
		t.Fatal("Sync error = nil, want context cancellation")
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if reloaded.LastNSU != 12 {
		t.Fatalf("LastNSU after interruption = %d, want 12", reloaded.LastNSU)
	}

	run := readLatestSyncRun(t, rawDB)
	if run.Status != "interrupted" || run.ToNSU != 12 {
		t.Fatalf("unexpected sync run: %+v", run)
	}
}

func TestSyncFailsOnProtocolRegressionWhenUltNSUGoesBackwards(t *testing.T) {
	svc, company, sqliteStore, rawDB := setupSyncTest(t, 20)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{UltNSU: 19, MaxNSU: 25, Docs: nil}},
		},
	}
	svc.apiClient = fetcher

	err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil)
	if err == nil {
		t.Fatal("Sync error = nil, want protocol failure")
	}

	reloaded, err := sqliteStore.GetCompany(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatalf("GetCompany: %v", err)
	}
	if reloaded.LastNSU != 20 {
		t.Fatalf("LastNSU after protocol failure = %d, want 20", reloaded.LastNSU)
	}

	run := readLatestSyncRun(t, rawDB)
	if run.Status != "failed" || run.ToNSU != 20 {
		t.Fatalf("unexpected sync run: %+v", run)
	}
}

func TestSyncPersistsEventEnvelopeAndUpdatesDocumentStatus(t *testing.T) {
	svc, company, sqliteStore, _ := setupSyncTest(t, 0)

	fetcher := &fakeFetcher{
		results: []fetchResult{
			{response: &adn.DocumentResponse{
				UltNSU: 2,
				MaxNSU: 2,
				Docs: []adn.DocumentEnvelope{
					makeDocumentEnvelope(1, "CHAVE-EVT", company.CNPJ, "22222222000122"),
					makeEventEnvelope(2, `<pedCancNFSe><infPedidoCanc><chNFSe>CHAVE-EVT</chNFSe><cMotivo>Cancelada</cMotivo><dhEvento>2026-06-04T13:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`),
				},
			}},
		},
	}
	svc.apiClient = fetcher

	if err := svc.Sync(context.Background(), company, testCredential(), "exact_certificate_cnpj", nil); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	doc, err := sqliteStore.GetDocumentByChave(context.Background(), "CHAVE-EVT")
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if doc == nil {
		t.Fatal("document not found after sync")
	}
	if doc.Status != "cancelada" {
		t.Fatalf("Status = %q, want cancelada", doc.Status)
	}

	events, err := sqliteStore.ListEventsByDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("ListEventsByDocument: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].RawXMLPath == "" || events[0].Type != "cancelamento" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

type syncRunSnapshot struct {
	Status         string
	ToNSU          int64
	DocumentsFound int
	ErrorsCount    int
}

func setupSyncTest(t *testing.T, lastNSU int64) (*SyncService, *nfse.Company, *store.SQLiteStore, *sql.DB) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "sync.db")

	sqliteStore, err := store.NewSQLiteStore(dbPath, true)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = rawDB.Close()
	})

	company := &nfse.Company{
		ID:           "company-1",
		CNPJ:         "12345678000199",
		CNPJRoot:     "12345678",
		Name:         "Sync Test Co",
		CredentialID: "cred-1",
		Environment:  "producao_restrita",
		LastNSU:      lastNSU,
	}
	if err := sqliteStore.CreateCredential(context.Background(), testCredential()); err != nil {
		t.Fatalf("CreateCredential: %v", err)
	}
	if err := sqliteStore.CreateCompany(context.Background(), company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	svc := NewSyncService(sqliteStore, &fakeFetcher{}, files.NewWriter(tempDir))
	return svc, company, sqliteStore, rawDB
}

func readLatestSyncRun(t *testing.T, db *sql.DB) syncRunSnapshot {
	t.Helper()

	var run syncRunSnapshot
	err := db.QueryRowContext(context.Background(),
		`SELECT status, COALESCE(to_nsu, 0), documents_found, errors_count
		 FROM sync_runs
		 ORDER BY started_at DESC, id DESC
		 LIMIT 1`,
	).Scan(&run.Status, &run.ToNSU, &run.DocumentsFound, &run.ErrorsCount)
	if err != nil {
		t.Fatalf("readLatestSyncRun: %v", err)
	}

	return run
}

func makeDocumentEnvelope(nsu int64, chave, prestadorCNPJ, tomadorCNPJ string) adn.DocumentEnvelope {
	xml := fmt.Sprintf(
		`<NFSe><infNFSe><chNFSe>%s</chNFSe><dhEmi>2026-06-04T12:00:00Z</dhEmi><compNFSe>2026-06-01</compNFSe><prest><CNPJ>%s</CNPJ><xNome>Prestador</xNome></prest><toma><CNPJ>%s</CNPJ><xNome>Tomador</xNome></toma><valores><vServ>100.00</vServ><vISS>5.00</vISS></valores></infNFSe></NFSe>`,
		chave,
		prestadorCNPJ,
		tomadorCNPJ,
	)

	return adn.DocumentEnvelope{
		NSU:           nsu,
		Schema:        "procNFSe_v1.00.xsd",
		XMLGZipBase64: gzipBase64(tobytes(xml)),
	}
}

func makeEventEnvelope(nsu int64, xml string) adn.DocumentEnvelope {
	return adn.DocumentEnvelope{
		NSU:           nsu,
		Schema:        "procEventoNFSe_v1.00.xsd",
		XMLGZipBase64: gzipBase64(tobytes(xml)),
	}
}

func testCredential() *nfse.Credential {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	return &nfse.Credential{
		ID:                "cred-1",
		Label:             "Cred Test",
		CertPath:          "cert.pfx",
		Environment:       "producao_restrita",
		OwnerCNPJ:         "12345678000199",
		OwnerCNPJRoot:     "12345678",
		FingerprintSHA256: "hash",
		SubjectName:       "CN=Cred Test",
		NotBefore:         &now,
		NotAfter:          &now,
		InspectedAt:       &now,
	}
}

func gzipBase64(data []byte) string {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(data)
	_ = gz.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func tobytes(s string) []byte {
	return []byte(s)
}
