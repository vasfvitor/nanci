package sync

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/nfse"
	dbstore "github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

type testHelper struct {
	t          *testing.T
	db         *sql.DB
	store      *Store
	company    *nfse.Company
	credential *nfse.Credential
}

func newTestHelper(t *testing.T) *testHelper {
	t.Helper()
	db := storetest.OpenTestDB(t)
	store := NewStore(db)

	cred := storetest.TestCredential("cred-1")
	if err := credential.NewStore(db).CreateCredential(context.Background(), cred); err != nil {
		t.Fatal(err)
	}

	company := storetest.TestCompany("comp-1", "12345678901234", nfse.EnvironmentProduction, cred)
	company.SyncStartPolicy = nfse.SyncStartPolicyAll
	company.SyncStartDate = nil
	if err := dbstore.NewCompanyRepository(db).CreateCompany(context.Background(), company); err != nil {
		t.Fatal(err)
	}

	return &testHelper{
		t:          t,
		db:         db,
		store:      store,
		company:    company,
		credential: cred,
	}
}

func (h *testHelper) setSyncState(nsu int64, lastFound *int64, streak int) {
	h.t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	var lastFoundVal sql.NullInt64
	if lastFound != nil {
		lastFoundVal = sql.NullInt64{Int64: *lastFound, Valid: true}
	}
	_, err := h.db.ExecContext(context.Background(), `
		INSERT INTO sync_state (
			company_id, environment, consultation_cnpj,
			last_checked_nsu, last_found_nsu, last_empty_streak,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(company_id, environment, consultation_cnpj) DO UPDATE SET
			last_checked_nsu = excluded.last_checked_nsu,
			last_found_nsu = excluded.last_found_nsu,
			last_empty_streak = excluded.last_empty_streak,
			updated_at = excluded.updated_at
	`, string(h.company.ID), string(h.company.Environment), h.company.CNPJ, nsu, lastFoundVal, streak, now, now)
	if err != nil {
		h.t.Fatalf("failed to set sync_state: %v", err)
	}
}

func (h *testHelper) assertSyncState(wantNSU int64, wantLastFound *int64, wantEmptyStreak int) {
	h.t.Helper()
	var nsu int64
	var lastFound sql.NullInt64
	var streak int
	err := h.db.QueryRowContext(context.Background(), `
		SELECT last_checked_nsu, last_found_nsu, last_empty_streak
		FROM sync_state
		WHERE company_id = ? AND environment = ? AND consultation_cnpj = ?
	`, string(h.company.ID), string(h.company.Environment), h.company.CNPJ).Scan(&nsu, &lastFound, &streak)
	if err != nil {
		h.t.Fatalf("failed to query sync_state: %v", err)
	}
	if nsu != wantNSU {
		h.t.Errorf("sync_state.last_checked_nsu = %d, want %d", nsu, wantNSU)
	}
	var wantLastFoundVal sql.NullInt64
	if wantLastFound != nil {
		wantLastFoundVal = sql.NullInt64{Int64: *wantLastFound, Valid: true}
	}
	if lastFound != wantLastFoundVal {
		h.t.Errorf("sync_state.last_found_nsu = %v, want %v", lastFound, wantLastFoundVal)
	}
	if streak != wantEmptyStreak {
		h.t.Errorf("sync_state.last_empty_streak = %d, want %d", streak, wantEmptyStreak)
	}
}

func (h *testHelper) assertSingleFinishRun(wantStatus nfse.SyncStatus, wantReason nfse.SyncStopReason) {
	h.t.Helper()
	rows, err := h.db.QueryContext(context.Background(), `
		SELECT status, stop_reason
		FROM sync_runs
		WHERE company_id = ?
	`, string(h.company.ID))
	if err != nil {
		h.t.Fatalf("failed to query sync_runs: %v", err)
	}
	defer rows.Close()

	var count int
	var status string
	var stopReason sql.NullString
	for rows.Next() {
		count++
		if err := rows.Scan(&status, &stopReason); err != nil {
			h.t.Fatal(err)
		}
	}
	if count != 1 {
		h.t.Fatalf("expected exactly 1 sync run, got %d", count)
	}
	if nfse.SyncStatus(status) != wantStatus {
		h.t.Errorf("sync_run status = %s, want %s", status, wantStatus)
	}
	var gotReason nfse.SyncStopReason
	if stopReason.Valid {
		gotReason = nfse.SyncStopReason(stopReason.String)
	}
	if gotReason != wantReason {
		h.t.Errorf("sync_run stop_reason = %s, want %s", gotReason, wantReason)
	}
}

func (h *testHelper) assertLatestSyncRun(wantStatus nfse.SyncStatus, wantReason nfse.SyncStopReason, wantDocsFound int) {
	h.t.Helper()
	var status string
	var stopReason sql.NullString
	var docsFound int
	err := h.db.QueryRowContext(context.Background(), `
		SELECT status, stop_reason, documents_found
		FROM sync_runs
		WHERE company_id = ?
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`, string(h.company.ID)).Scan(&status, &stopReason, &docsFound)
	if err != nil {
		h.t.Fatalf("failed to query sync_runs: %v", err)
	}
	if nfse.SyncStatus(status) != wantStatus {
		h.t.Errorf("sync_run status = %s, want %s", status, wantStatus)
	}
	var gotReason nfse.SyncStopReason
	if stopReason.Valid {
		gotReason = nfse.SyncStopReason(stopReason.String)
	}
	if gotReason != wantReason {
		h.t.Errorf("sync_run stop_reason = %s, want %s", gotReason, wantReason)
	}
	if docsFound != wantDocsFound {
		h.t.Errorf("sync_run documents_found = %d, want %d", docsFound, wantDocsFound)
	}
}

func (h *testHelper) insertLocalDocument(accessKey string) {
	h.t.Helper()
	doc := nfse.Document{
		ID:                 nfse.DocumentID("doc-" + accessKey[:min(8, len(accessKey))]),
		ChaveAcesso:        nfse.AccessKey(accessKey),
		IssueDate:          time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC),
		Competence:         "2026-06",
		PrestadorCNPJ:      "99887766000155",
		PrestadorName:      "Provider",
		TomadorCNPJ:        "12345678901234",
		TomadorName:        "Customer",
		ServiceValue:       10000,
		Status:             nfse.DocumentStatusNormal,
		LayoutVersion:      "1.01",
		XMLPath:            "hash-" + accessKey[:min(8, len(accessKey))] + ".xml",
		RawHash:            "hash-" + accessKey[:min(8, len(accessKey))],
		NFSeNumber:         "123",
		ServiceDescription: "service",
	}
	_, err := h.store.ApplyDocument(context.Background(), nfse.ApplyDocumentParams{
		Document: doc,
		Participation: nfse.CompanyParticipation{
			CompanyRole:      nfse.CompanyRoleTomada,
			VisibilityReason: "exact_tomador",
		},
		CompanyID: h.company.ID,
		NSU:       1,
	})
	if err != nil {
		h.t.Fatalf("failed to insert local document: %v", err)
	}
}

func (h *testHelper) assertDocumentsCount(wantCount int) {
	h.t.Helper()
	var count int
	err := h.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM documents`).Scan(&count)
	if err != nil {
		h.t.Fatalf("failed to count documents: %v", err)
	}
	if count != wantCount {
		h.t.Errorf("documents count = %d, want %d", count, wantCount)
	}
}

func (h *testHelper) assertEventsCount(wantCount int) {
	h.t.Helper()
	var count int
	err := h.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM events`).Scan(&count)
	if err != nil {
		h.t.Fatalf("failed to count events: %v", err)
	}
	if count != wantCount {
		h.t.Errorf("events count = %d, want %d", count, wantCount)
	}
}

func (h *testHelper) getAppliedNSUOrder() []int64 {
	h.t.Helper()
	rows, err := h.db.QueryContext(context.Background(), `SELECT first_seen_nsu FROM company_documents ORDER BY rowid`)
	if err != nil {
		h.t.Fatalf("failed to query apply order: %v", err)
	}
	defer rows.Close()
	var nsus []int64
	for rows.Next() {
		var nsu int64
		if err := rows.Scan(&nsu); err != nil {
			h.t.Fatal(err)
		}
		nsus = append(nsus, nsu)
	}
	return nsus
}

func (h *testHelper) assertInitialSyncCompleted(wantCompleted bool) {
	h.t.Helper()
	var completedAt sql.NullString
	err := h.db.QueryRowContext(context.Background(), `
		SELECT initial_sync_completed_at
		FROM companies
		WHERE id = ?
	`, string(h.company.ID)).Scan(&completedAt)
	if err != nil {
		h.t.Fatalf("failed to query initial_sync_completed_at: %v", err)
	}
	if (completedAt.Valid) != wantCompleted {
		h.t.Errorf("initial_sync_completed_at valid = %t, want %t", completedAt.Valid, wantCompleted)
	}
}

func (h *testHelper) markInitialSyncDone(t *testing.T, doneAt time.Time) {
	h.t.Helper()
	_, err := h.db.ExecContext(context.Background(), `
		UPDATE companies
		SET initial_sync_completed_at = ?
		WHERE id = ?
	`, doneAt.Format(time.RFC3339), string(h.company.ID))
	if err != nil {
		h.t.Fatalf("failed to mark initial sync done: %v", err)
	}
	h.company.InitialSyncDoneAt = &doneAt
}

type mockFetcher struct {
	handler  func(adn.DistributionRequest) (*adn.DocumentResponse, error)
	requests []adn.DistributionRequest
}

func (m *mockFetcher) FetchDocuments(ctx context.Context, req adn.DistributionRequest) (*adn.DocumentResponse, error) {
	m.requests = append(m.requests, req)
	if m.handler != nil {
		return m.handler(req)
	}
	return &adn.DocumentResponse{}, nil
}

type mockXMLStore struct {
	stored map[string][]byte
}

func (m *mockXMLStore) Store(hash string, data []byte) error {
	if m.stored == nil {
		m.stored = make(map[string][]byte)
	}
	m.stored[hash] = data
	return nil
}

func (m *mockXMLStore) Get(hash string) ([]byte, error) {
	if data, ok := m.stored[hash]; ok {
		return data, nil
	}
	return nil, errors.New("not found")
}

func TestSyncServiceSuccessFinishesOnceAsCompleted(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)
	h.insertLocalDocument("12345678901234567890123456789012345678901234567890")

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch, got %d", len(fetcher.requests))
	}
	h.assertSyncState(0, nil, 1)
	h.assertSingleFinishRun(nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit)
}

func TestSyncServiceNormalModeUsesPersistedCursorWithoutRevisit(t *testing.T) {
	h := newTestHelper(t)
	lastFound := int64(29)
	h.setSyncState(29, &lastFound, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch, got %d", len(fetcher.requests))
	}
	if fetcher.requests[0].LastNSU != 29 {
		t.Fatalf("requested LastNSU = %d, want 29", fetcher.requests[0].LastNSU)
	}
	h.assertSyncState(29, &lastFound, 1)
}

func TestSyncServiceFetchFailureFinishesOnceAsFailed(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)
	h.insertLocalDocument("12345678901234567890123456789012345678901234567890")

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return nil, errors.New("api error")
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	h.assertSingleFinishRun(nfse.SyncStatusFailed, nfse.SyncStopReasonFetchError)
}

func TestSyncServiceCancellationFinishesOnceAsInterrupted(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	ctx, cancel := context.WithCancel(context.Background())
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			cancel()
			return nil, context.Canceled
		},
	}

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	err := svc.Sync(ctx, h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	h.assertSingleFinishRun(nfse.SyncStatusInterrupted, nfse.SyncStopReasonContextCanceled)
}

func TestSyncServiceProcessingFailurePersistsConsultedCheckpointBeforeFailing(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 0:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					},
				}, nil
			case 1:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 2, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, "<NFSe>")},
					},
				}, nil
			default:
				return &adn.DocumentResponse{}, nil
			}
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	lastFound := int64(1)
	h.assertSyncState(1, &lastFound, 0)
	h.assertSingleFinishRun(nfse.SyncStatusFailed, nfse.SyncStopReasonProcessError)
}

func TestSyncServiceProcessesEventFromTipoEventoMetadata(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)
	h.insertLocalDocument("12345678901234567890123456789012345678901234567890")

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{
							NSU:           1,
							Schema:        "procNFSe_v1.00.xsd",
							XMLGZipBase64: mustEncodeGzipBase64(t, `<pedCancNFSe><infPedidoCanc><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><cMotivo>Erro emissao</cMotivo><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`),
							EventType:     "CANCELAMENTO",
						},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertEventsCount(1)
	h.assertDocumentsCount(1)
}

func TestSyncServiceProcessesBatchInAscendingNSUOrder(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 3, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567893"))},
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567891"))},
						{NSU: 2, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567892"))},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertDocumentsCount(3)
	order := h.getAppliedNSUOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 applied documents, got %v", order)
	}
	if order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("documents not applied in ascending NSU order: %v", order)
	}
}

func TestSyncServiceAdvancesCursorAcrossBatchesWithoutIncrementingRequestNSU(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 0:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567891"))},
						{NSU: 2, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567892"))},
						{NSU: 29, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567893"))},
					},
				}, nil
			case 29:
				return &adn.DocumentResponse{}, nil
			default:
				t.Fatalf("unexpected LastNSU request %d", req.LastNSU)
				return nil, errors.New("unreachable")
			}
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 2 {
		t.Fatalf("expected 2 fetches, got %d", len(fetcher.requests))
	}
	if fetcher.requests[0].LastNSU != 0 || fetcher.requests[1].LastNSU != 29 {
		t.Fatalf("unexpected LastNSU sequence: %+v", fetcher.requests)
	}

	h.assertSyncState(29, storetest.Int64Ptr(29), 1)
	h.assertLatestSyncRun(nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit, 3)
}

func TestSyncServiceSkipsStaleDocumentsAndAdvancesToFreshNSU(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(29, storetest.Int64Ptr(29), 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 29:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 28, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567891"))},
						{NSU: 29, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567892"))},
						{NSU: 30, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "12345678901234567890123456789012345678901234567890", "12345678901234567890123456789012345678901234567893"))},
					},
				}, nil
			case 30:
				return &adn.DocumentResponse{}, nil
			default:
				t.Fatalf("unexpected LastNSU request %d", req.LastNSU)
				return nil, errors.New("unreachable")
			}
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertDocumentsCount(1) // Only NSU 30 is inserted
	h.assertSyncState(30, storetest.Int64Ptr(30), 1)
}

func TestSyncServiceAdvancesCursorOnDuplicateAboveCurrentCursor(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(29, storetest.Int64Ptr(29), 0)
	h.insertLocalDocument("12345678901234567890123456789012345678901234567890")

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 29:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 30, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					},
				}, nil
			case 30:
				return &adn.DocumentResponse{}, nil
			default:
				t.Fatalf("unexpected LastNSU request %d", req.LastNSU)
				return nil, errors.New("unreachable")
			}
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 2 {
		t.Fatalf("expected 2 fetches, got %d", len(fetcher.requests))
	}
	if fetcher.requests[1].LastNSU != 30 {
		t.Fatalf("second LastNSU = %d, want 30", fetcher.requests[1].LastNSU)
	}
	h.assertSyncState(30, storetest.Int64Ptr(30), 1)
	h.assertLatestSyncRun(nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit, 0)
}

func TestSyncServiceSkipsDocumentBeforeInitialSyncCutoffAndAdvancesCursor(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 0:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "2026-06", "2024-06"))},
					},
				}, nil
			case 1:
				return &adn.DocumentResponse{}, nil
			default:
				t.Fatalf("unexpected LastNSU request %d", req.LastNSU)
				return nil, errors.New("unreachable")
			}
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	xmlStore := &mockXMLStore{}
	svc := NewSyncService(h.store, fetcher, xmlStore, discardLogger())
	cutoff := mustDate(t, "2025-01-01")
	h.company.SyncStartPolicy = nfse.SyncStartPolicySinceDate
	h.company.SyncStartDate = &cutoff

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertDocumentsCount(0)
	if len(xmlStore.stored) != 0 {
		t.Fatalf("expected no stored XML, got %d", len(xmlStore.stored))
	}
	h.assertSyncState(1, storetest.Int64Ptr(1), 1)
	h.assertInitialSyncCompleted(true)
}

func TestSyncServiceContinuesBootstrapAfterPartialRunWithAdvancedCursor(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(500, storetest.Int64Ptr(500), 0)

	fetcher := &mockFetcher{}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	cutoff := mustDate(t, "2025-01-01")
	h.company.SyncStartPolicy = nfse.SyncStartPolicySinceDate
	h.company.SyncStartDate = &cutoff

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	fetcher.handler = func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
		if req.LastNSU == 500 {
			return &adn.DocumentResponse{
				Docs: []adn.DocumentEnvelope{
					{NSU: 501, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "2026-06", "2024-06"))},
				},
			}, nil
		}
		return &adn.DocumentResponse{}, nil
	}

	err := svc.Sync(ctx, h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, func(e nfse.ProgressEvent) {
		cancel()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled after partial bootstrap, got %v", err)
	}

	h.assertSyncState(501, storetest.Int64Ptr(501), 0)
	h.assertDocumentsCount(0)
}

func TestSyncServiceProcessesOldDocumentAfterInitialBootstrapCompleted(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)
	doneAt := mustDate(t, "2026-06-21")
	h.markInitialSyncDone(t, doneAt)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "2026-06", "2024-06"))},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	cutoff := mustDate(t, "2025-01-01")
	h.company.SyncStartPolicy = nfse.SyncStartPolicySinceDate
	h.company.SyncStartDate = &cutoff

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertDocumentsCount(1)
}

func TestSyncServiceSkipsEventWithoutLocalDocument(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{
							NSU:           1,
							Schema:        "procNFSe_v1.00.xsd",
							XMLGZipBase64: mustEncodeGzipBase64(t, `<pedCancNFSe><infPedidoCanc><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><cMotivo>Erro emissao</cMotivo><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`),
							EventType:     "CANCELAMENTO",
						},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}
	h.assertEventsCount(0)
	h.assertSyncState(1, storetest.Int64Ptr(1), 1)
}

func TestSyncServiceStopsOnNonAdvancingBatch(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(29, storetest.Int64Ptr(29), 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{
				Docs: []adn.DocumentEnvelope{
					{NSU: 28, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					{NSU: 29, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
				},
			}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch, got %d", len(fetcher.requests))
	}
	h.assertDocumentsCount(0)
	h.assertSyncState(29, storetest.Int64Ptr(29), 0)
}

func TestSyncServiceBreaksGracefullyOnEmptyList(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{
				Docs: []adn.DocumentEnvelope{},
			}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertSingleFinishRun(nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit)
}

func TestSyncServiceApplyDocumentAndProgressIsAtomic(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(h.store, fetcher, &mockXMLStore{}, discardLogger())

	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeFirstSetup, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	h.assertDocumentsCount(1)
	h.assertSyncState(1, storetest.Int64Ptr(1), 1)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil)) //nolint:sloglint
}

func mustEncodeGzipBase64(t *testing.T, xml string) string {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(xml)); err != nil {
		t.Fatalf("failed to gzip xml: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

const validDocumentXML = `<?xml version="1.0" encoding="UTF-8"?>
<NFSe xmlns="http://www.sped.fazenda.gov.br/nfse">
  <infNFSe versao="1.00">
    <chNFSe>12345678901234567890123456789012345678901234567890</chNFSe>
    <nNFSe>10001</nNFSe>
    <dhEmi>2026-06-07T10:00:00-03:00</dhEmi>
    <compNFSe>2026-06</compNFSe>
    <prest>
      <CNPJ>12345678000100</CNPJ>
      <xNome>Prestador Teste</xNome>
    </prest>
    <toma>
      <CNPJ>98765432000199</CNPJ>
      <xNome>Tomador Teste</xNome>
    </toma>
    <valores>
      <vServ>1500.50</vServ>
      <vISS>30.00</vISS>
      <vIRRF>10.00</vIRRF>
      <vINSS>5.00</vINSS>
      <vPIS>9.75</vPIS>
      <vCOFINS>45.00</vCOFINS>
      <vCSLL>15.00</vCSLL>
    </valores>
    <xDescServ>Consultoria e development</xDescServ>
  </infNFSe>
</NFSe>`
