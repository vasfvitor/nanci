package syncrun

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func int64Ptr(v int64) *int64 { return &v }

type mockSyncRepo struct {
	state                         *nfse.SyncState
	startRunParams                []nfse.StartRunParams
	persistParams                 []nfse.PersistSyncProgressParams
	finishRunParams               []nfse.FinishRunParams
	applyDocParams                []nfse.ApplyDocumentParams
	applyEventParams              []nfse.ApplyEventParams
	applyDocAndProgressParams     []nfse.ApplyDocumentAndProgressParams
	applyEventAndProgressParams   []nfse.ApplyEventAndProgressParams
	applyDocOutcomeByNSU          map[int64]nfse.ApplyOutcome
	applyEventOutcomeByNSU        map[int64]nfse.ApplyOutcome
	localDocumentByAccessKey      map[string]bool
	markInitialSyncCompletedCalls int
}

func (m *mockSyncRepo) GetOrCreateState(context.Context, nfse.GetOrCreateSyncStateParams) (*nfse.SyncState, error) {
	if m.state == nil {
		m.state = &nfse.SyncState{}
	}
	stateCopy := *m.state
	return &stateCopy, nil
}

func (m *mockSyncRepo) StartRun(ctx context.Context, p nfse.StartRunParams) (nfse.SyncRun, error) {
	m.startRunParams = append(m.startRunParams, p)
	return nfse.SyncRun{ID: "run-1", Status: nfse.SyncStatusRunning}, nil
}

func (m *mockSyncRepo) PersistProgress(ctx context.Context, p nfse.PersistSyncProgressParams) error {
	m.persistParams = append(m.persistParams, p)
	if m.state == nil {
		m.state = &nfse.SyncState{}
	}
	m.state.LastProcessedNSU = p.LastProcessedNSU
	m.state.LastFoundNSU = p.LastFoundNSU
	m.state.LastEmptyStreak = p.LastEmptyStreak
	return nil
}

func (m *mockSyncRepo) FinishRun(ctx context.Context, p nfse.FinishRunParams) error {
	m.finishRunParams = append(m.finishRunParams, p)
	return nil
}

func (m *mockSyncRepo) ApplyDocument(ctx context.Context, p nfse.ApplyDocumentParams) (nfse.ApplyOutcome, error) {
	m.applyDocParams = append(m.applyDocParams, p)
	return m.documentOutcome(p.NSU), nil
}

func (m *mockSyncRepo) ApplyEvent(ctx context.Context, p nfse.ApplyEventParams) (nfse.ApplyOutcome, error) {
	m.applyEventParams = append(m.applyEventParams, p)
	return m.eventOutcome(p.NSU), nil
}

func (m *mockSyncRepo) ApplyDocumentAndProgress(ctx context.Context, p nfse.ApplyDocumentAndProgressParams) (nfse.ApplyOutcome, error) {
	m.applyDocAndProgressParams = append(m.applyDocAndProgressParams, p)
	outcome := m.documentOutcome(p.DocumentParams.NSU)
	progressParams := p.ProgressParams
	if outcome.Inserted {
		progressParams.DocumentsFound++
	}
	return outcome, m.PersistProgress(ctx, progressParams)
}

func (m *mockSyncRepo) ApplyEventAndProgress(ctx context.Context, p nfse.ApplyEventAndProgressParams) (nfse.ApplyOutcome, error) {
	m.applyEventAndProgressParams = append(m.applyEventAndProgressParams, p)
	outcome := m.eventOutcome(p.EventParams.NSU)
	return outcome, m.PersistProgress(ctx, p.ProgressParams)
}

func (m *mockSyncRepo) LatestSyncSnapshot(context.Context, nfse.CompanyID, nfse.Environment, string) (nfse.SyncSnapshot, error) {
	return nfse.SyncSnapshot{}, nil
}

func (m *mockSyncRepo) ResetSyncState(context.Context, nfse.ResetSyncStateParams) error {
	return nil
}

func (m *mockSyncRepo) HasSyncState(context.Context, nfse.HasSyncStateParams) (bool, error) {
	return m.state != nil, nil
}

func (m *mockSyncRepo) MarkInitialSyncCompleted(context.Context, nfse.CompanyID) error {
	m.markInitialSyncCompletedCalls++
	return nil
}

func (m *mockSyncRepo) CompanyDocumentExistsByAccessKey(_ context.Context, _ nfse.CompanyID, chave string) (bool, error) {
	return m.localDocumentByAccessKey[chave], nil
}

func (m *mockSyncRepo) documentOutcome(nsu int64) nfse.ApplyOutcome {
	if m.applyDocOutcomeByNSU != nil {
		if outcome, ok := m.applyDocOutcomeByNSU[nsu]; ok {
			return outcome
		}
	}
	return nfse.ApplyOutcome{Inserted: true}
}

func (m *mockSyncRepo) eventOutcome(nsu int64) nfse.ApplyOutcome {
	if m.applyEventOutcomeByNSU != nil {
		if outcome, ok := m.applyEventOutcomeByNSU[nsu]; ok {
			return outcome
		}
	}
	return nfse.ApplyOutcome{Inserted: true}
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
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
		localDocumentByAccessKey: map[string]bool{
			"12345678901234567890123456789012345678901234567890": true,
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch, got %d", len(fetcher.requests))
	}
	if got := repo.persistParams[len(repo.persistParams)-1].LastProcessedNSU; got != 0 {
		t.Fatalf("last processed nsu = %d, want 0", got)
	}
	assertSingleFinishRun(t, repo, nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit)
}

func TestSyncServiceNormalModeUsesPersistedCursorWithoutRevisit(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
			LastProcessedNSU: 29,
			LastFoundNSU:     int64Ptr(29),
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch, got %d", len(fetcher.requests))
	}
	if fetcher.requests[0].LastNSU != 29 {
		t.Fatalf("requested LastNSU = %d, want 29", fetcher.requests[0].LastNSU)
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 29 {
		t.Fatalf("last processed nsu = %d, want 29", last.LastProcessedNSU)
	}
	if last.DocumentsFound != 0 {
		t.Fatalf("documents found = %d, want 0", last.DocumentsFound)
	}
}

func TestSyncServiceFetchFailureFinishesOnceAsFailed(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
		localDocumentByAccessKey: map[string]bool{
			"12345678901234567890123456789012345678901234567890": true,
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return nil, errors.New("api error")
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	assertSingleFinishRun(t, repo, nfse.SyncStatusFailed, nfse.SyncStopReasonFetchError)
}

func TestSyncServiceCancellationFinishesOnceAsInterrupted(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
	fetcher := &mockFetcher{}

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.Sync(ctx, company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	assertSingleFinishRun(t, repo, nfse.SyncStatusInterrupted, nfse.SyncStopReasonContextCanceled)
}

func TestSyncServiceProcessingFailurePersistsConsultedCheckpointBeforeFailing(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	if len(repo.persistParams) < 2 {
		t.Fatalf("expected at least 2 progress writes, got %d", len(repo.persistParams))
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 1 {
		t.Fatalf("last persisted checkpoint = %d, want 1", last.LastProcessedNSU)
	}
	assertSingleFinishRun(t, repo, nfse.SyncStatusFailed, nfse.SyncStopReasonProcessError)
}

func TestSyncServiceProcessesEventFromTipoEventoMetadata(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
		localDocumentByAccessKey: map[string]bool{
			"12345678901234567890123456789012345678901234567890": true,
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyEventAndProgressParams) != 1 {
		t.Fatalf("expected 1 event to be applied atomically, got %d", len(repo.applyEventAndProgressParams))
	}
	if len(repo.applyDocAndProgressParams) != 0 {
		t.Fatalf("expected no documents to be applied, got %d", len(repo.applyDocAndProgressParams))
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil)) //nolint:sloglint
}

func assertSingleFinishRun(t *testing.T, repo *mockSyncRepo, wantStatus nfse.SyncStatus, wantReason nfse.SyncStopReason) {
	t.Helper()
	if len(repo.finishRunParams) != 1 {
		t.Fatalf("expected exactly 1 finish call, got %d", len(repo.finishRunParams))
	}
	if repo.finishRunParams[0].Status != wantStatus {
		t.Fatalf("finish status = %s, want %s", repo.finishRunParams[0].Status, wantStatus)
	}
	if repo.finishRunParams[0].StopReason != wantReason {
		t.Fatalf("finish stop_reason = %s, want %s", repo.finishRunParams[0].StopReason, wantReason)
	}
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
    <xDescServ>Consultoria e desenvolvimento</xDescServ>
  </infNFSe>
</NFSe>`

func TestSyncServiceProcessesBatchInAscendingNSUOrder(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 3, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
						{NSU: 2, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyDocAndProgressParams) != 3 {
		t.Fatalf("expected 3 documents applied, got %d", len(repo.applyDocAndProgressParams))
	}

	if repo.applyDocAndProgressParams[0].DocumentParams.NSU != 1 || repo.applyDocAndProgressParams[1].DocumentParams.NSU != 2 || repo.applyDocAndProgressParams[2].DocumentParams.NSU != 3 {
		t.Fatalf("documents not applied in ascending NSU order")
	}
}

func TestSyncServiceAdvancesCursorAcrossBatchesWithoutIncrementingRequestNSU(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 0:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
						{NSU: 2, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
						{NSU: 29, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 2 {
		t.Fatalf("expected 2 fetches, got %d", len(fetcher.requests))
	}
	if fetcher.requests[0].LastNSU != 0 || fetcher.requests[1].LastNSU != 29 {
		t.Fatalf("unexpected LastNSU sequence: %+v", fetcher.requests)
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 29 {
		t.Fatalf("last processed nsu = %d, want 29", last.LastProcessedNSU)
	}
	if last.DocumentsFound != 3 {
		t.Fatalf("documents found = %d, want 3", last.DocumentsFound)
	}
}

func TestSyncServiceSkipsStaleDocumentsAndAdvancesToFreshNSU(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
			LastProcessedNSU: 29,
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			switch req.LastNSU {
			case 29:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 28, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
						{NSU: 29, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyDocAndProgressParams) != 1 {
		t.Fatalf("expected 1 applied document, got %d", len(repo.applyDocAndProgressParams))
	}
	if repo.applyDocAndProgressParams[0].DocumentParams.NSU != 30 {
		t.Fatalf("applied NSU = %d, want 30", repo.applyDocAndProgressParams[0].DocumentParams.NSU)
	}
}

func TestSyncServiceAdvancesCursorOnDuplicateAboveCurrentCursor(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
			LastProcessedNSU: 29,
		},
		applyDocOutcomeByNSU: map[int64]nfse.ApplyOutcome{
			30: {Inserted: false},
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 2 {
		t.Fatalf("expected 2 fetches, got %d", len(fetcher.requests))
	}
	if fetcher.requests[1].LastNSU != 30 {
		t.Fatalf("second LastNSU = %d, want 30", fetcher.requests[1].LastNSU)
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 30 {
		t.Fatalf("last processed nsu = %d, want 30", last.LastProcessedNSU)
	}
	if last.DocumentsFound != 0 {
		t.Fatalf("documents found = %d, want 0", last.DocumentsFound)
	}
}

func TestSyncServiceSkipsDocumentBeforeInitialSyncCutoffAndAdvancesCursor(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
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
	svc := NewSyncService(repo, fetcher, xmlStore, discardLogger())
	cutoff := mustDate(t, "2025-01-01")
	company := &nfse.Company{
		ID:              "comp-1",
		CNPJ:            "12345678901234",
		Environment:     nfse.EnvironmentProduction,
		SyncStartPolicy: nfse.SyncStartPolicySinceDate,
		SyncStartDate:   &cutoff,
	}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyDocAndProgressParams) != 0 {
		t.Fatalf("expected no applied documents, got %d", len(repo.applyDocAndProgressParams))
	}
	if len(xmlStore.stored) != 0 {
		t.Fatalf("expected no stored XML, got %d", len(xmlStore.stored))
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 1 {
		t.Fatalf("last processed nsu = %d, want 1", last.LastProcessedNSU)
	}
	if repo.markInitialSyncCompletedCalls != 1 {
		t.Fatalf("mark initial sync completed calls = %d, want 1", repo.markInitialSyncCompletedCalls)
	}
}

func TestSyncServiceContinuesBootstrapAfterPartialRunWithAdvancedCursor(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
			LastProcessedNSU: 500,
		},
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU != 500 {
				t.Fatalf("requested LastNSU = %d, want 500", req.LastNSU)
			}
			return &adn.DocumentResponse{
				Docs: []adn.DocumentEnvelope{
					{NSU: 501, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "2026-06", "2024-06"))},
				},
			}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	cutoff := mustDate(t, "2025-01-01")
	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{
		ID:              "comp-1",
		CNPJ:            "12345678901234",
		Environment:     nfse.EnvironmentProduction,
		SyncStartPolicy: nfse.SyncStartPolicySinceDate,
		SyncStartDate:   &cutoff,
	}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	ctx, cancel := context.WithCancel(context.Background())
	fetcher.handler = func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
		if req.LastNSU == 500 {
			cancel()
			return &adn.DocumentResponse{
				Docs: []adn.DocumentEnvelope{
					{NSU: 501, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, strings.ReplaceAll(validDocumentXML, "2026-06", "2024-06"))},
				},
			}, nil
		}
		return &adn.DocumentResponse{}, nil
	}

	err := svc.Sync(ctx, company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled after partial bootstrap, got %v", err)
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 501 {
		t.Fatalf("last processed nsu = %d, want 501", last.LastProcessedNSU)
	}
	if len(repo.applyDocAndProgressParams) != 0 {
		t.Fatalf("expected policy skip during partial bootstrap, got %d docs", len(repo.applyDocAndProgressParams))
	}
}

func TestSyncServiceProcessesOldDocumentAfterInitialBootstrapCompleted(t *testing.T) {
	doneAt := mustDate(t, "2026-06-21")
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
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
	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{
		ID:                "comp-1",
		CNPJ:              "12345678901234",
		Environment:       nfse.EnvironmentProduction,
		SyncStartPolicy:   nfse.SyncStartPolicySinceDate,
		SyncStartDate:     &cutoff,
		InitialSyncDoneAt: &doneAt,
	}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}
	if len(repo.applyDocAndProgressParams) != 1 {
		t.Fatalf("expected old document to be applied after bootstrap, got %d", len(repo.applyDocAndProgressParams))
	}
}

func TestSyncServiceSkipsEventWithoutLocalDocument(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}
	if len(repo.applyEventAndProgressParams) != 0 {
		t.Fatalf("expected no events applied, got %d", len(repo.applyEventAndProgressParams))
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 1 {
		t.Fatalf("last processed nsu = %d, want 1", last.LastProcessedNSU)
	}
}

func TestSyncServiceStopsOnNonAdvancingBatch(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
			LastProcessedNSU: 29,
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch, got %d", len(fetcher.requests))
	}
	if len(repo.applyDocAndProgressParams) != 0 {
		t.Fatalf("expected no applied documents, got %d", len(repo.applyDocAndProgressParams))
	}
	last := repo.persistParams[len(repo.persistParams)-1]
	if last.LastProcessedNSU != 29 {
		t.Fatalf("last processed nsu = %d, want 29", last.LastProcessedNSU)
	}
}

func TestSyncServiceBreaksGracefullyOnEmptyList(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	assertSingleFinishRun(t, repo, nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit)
}

func TestSyncServiceApplyDocumentAndProgressIsAtomic(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
		},
	}
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, discardLogger())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeFirstSetup, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyDocAndProgressParams) != 1 {
		t.Fatalf("expected atomic ApplyDocumentAndProgress to be called 1 time, got %d", len(repo.applyDocAndProgressParams))
	}

	if len(repo.applyDocParams) != 0 {
		t.Fatalf("expected non-atomic ApplyDocument to NOT be called, but got %d calls", len(repo.applyDocParams))
	}
}
