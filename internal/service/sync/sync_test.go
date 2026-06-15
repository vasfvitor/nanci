package syncservice

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"testing"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/nfse"
)

type mockSyncRepo struct {
	state            *nfse.SyncState
	startRunParams   []nfse.StartRunParams
	persistParams    []nfse.PersistSyncProgressParams
	finishRunParams  []nfse.FinishRunParams
	applyDocParams   []nfse.ApplyDocumentParams
	applyEventParams []nfse.ApplyEventParams
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
	m.state.LastCheckedNSU = p.LastCheckedNSU
	m.state.LastFoundNSU = p.LastFoundNSU
	m.state.LastFoundNSUValid = p.LastFoundNSUValid
	m.state.LastEmptyStreak = p.LastEmptyStreak
	return nil
}

func (m *mockSyncRepo) FinishRun(ctx context.Context, p nfse.FinishRunParams) error {
	m.finishRunParams = append(m.finishRunParams, p)
	return nil
}

func (m *mockSyncRepo) ApplyDocument(ctx context.Context, p nfse.ApplyDocumentParams) error {
	m.applyDocParams = append(m.applyDocParams, p)
	return nil
}

func (m *mockSyncRepo) ApplyEvent(ctx context.Context, p nfse.ApplyEventParams) error {
	m.applyEventParams = append(m.applyEventParams, p)
	return nil
}

func (m *mockSyncRepo) LatestSyncSnapshot(context.Context, nfse.CompanyID, nfse.Environment, string) (nfse.SyncSnapshot, error) {
	return nfse.SyncSnapshot{}, nil
}

func (m *mockSyncRepo) ResetSyncState(context.Context, nfse.ResetSyncStateParams) error {
	return nil
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
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			return &adn.DocumentResponse{}, nil
		},
	}

	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != normalEmptyLimit {
		t.Fatalf("expected %d fetches, got %d", normalEmptyLimit, len(fetcher.requests))
	}
	if got := repo.persistParams[len(repo.persistParams)-1].LastCheckedNSU; got != int64(normalEmptyLimit) {
		t.Fatalf("last checked nsu = %d, want %d", got, normalEmptyLimit)
	}
	assertSingleFinishRun(t, repo, nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit)
}

func TestSyncServiceFetchFailureFinishesOnceAsFailed(t *testing.T) {
	repo := &mockSyncRepo{
		state: &nfse.SyncState{
			CompanyID:        "comp-1",
			Environment:      nfse.EnvironmentProduction,
			ConsultationCNPJ: "12345678901234",
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())
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
			case 1:
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					},
				}, nil
			case 2:
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())
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
	if last.LastCheckedNSU != 2 {
		t.Fatalf("last persisted checkpoint = %d, want 2", last.LastCheckedNSU)
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
	}
	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 1 {
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

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", Environment: nfse.EnvironmentProduction}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyEventParams) != 1 {
		t.Fatalf("expected 1 event to be applied, got %d", len(repo.applyEventParams))
	}
	if len(repo.applyDocParams) != 0 {
		t.Fatalf("expected no documents to be applied, got %d", len(repo.applyDocParams))
	}
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
