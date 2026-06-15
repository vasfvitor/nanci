package syncservice

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/nfse"
)

type mockSyncRepo struct {
	startRunParams   []nfse.StartRunParams
	advanceParams    []nfse.AdvanceCheckpointParams
	finishRunParams  []nfse.FinishRunParams
	applyDocParams   []nfse.ApplyDocumentParams
	applyEventParams []nfse.ApplyEventParams
	applyDocumentErr func(nfse.ApplyDocumentParams) error
	applyEventErr    func(nfse.ApplyEventParams) error
}

func (m *mockSyncRepo) StartRun(ctx context.Context, p nfse.StartRunParams) (nfse.SyncRun, error) {
	m.startRunParams = append(m.startRunParams, p)
	return nfse.SyncRun{ID: "run-1", Status: "running"}, nil
}

func (m *mockSyncRepo) AdvanceCheckpoint(ctx context.Context, p nfse.AdvanceCheckpointParams) error {
	m.advanceParams = append(m.advanceParams, p)
	return nil
}

func (m *mockSyncRepo) FinishRun(ctx context.Context, p nfse.FinishRunParams) error {
	m.finishRunParams = append(m.finishRunParams, p)
	return nil
}

func (m *mockSyncRepo) ApplyDocument(ctx context.Context, p nfse.ApplyDocumentParams) error {
	m.applyDocParams = append(m.applyDocParams, p)
	if m.applyDocumentErr != nil {
		return m.applyDocumentErr(p)
	}
	return nil
}

func (m *mockSyncRepo) ApplyEvent(ctx context.Context, p nfse.ApplyEventParams) error {
	m.applyEventParams = append(m.applyEventParams, p)
	if m.applyEventErr != nil {
		return m.applyEventErr(p)
	}
	return nil
}

type mockFetcher struct {
	requests  []adn.DistributionRequest
	responses []func() (*adn.DocumentResponse, error)
	callCount int
}

func (m *mockFetcher) FetchDocuments(ctx context.Context, req adn.DistributionRequest) (*adn.DocumentResponse, error) {
	m.requests = append(m.requests, req)
	if m.callCount < len(m.responses) {
		resp, err := m.responses[m.callCount]()
		m.callCount++
		return resp, err
	}
	return nil, errors.New("unexpected call")
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

func TestSyncService_Loop(t *testing.T) {
	repo := &mockSyncRepo{}
	fetcher := &mockFetcher{
		responses: []func() (*adn.DocumentResponse, error){
			func() (*adn.DocumentResponse, error) {
				return &adn.DocumentResponse{
					UltNSU: 10,
					MaxNSU: 10,
					Docs:   nil,
				}, nil
			},
		},
	}
	xmlStore := &mockXMLStore{}
	logger := slog.Default()

	svc := NewSyncService(repo, fetcher, xmlStore, logger)

	company := &nfse.Company{
		ID:      "comp-1",
		CNPJ:    "12345678901234",
		LastNSU: 0,
	}
	credential := &nfse.Credential{
		ID:        "cred-1",
		OwnerCNPJ: "12345678901234",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := svc.Sync(ctx, company, credential, "exact", nil)
	if err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Errorf("expected 1 fetch call, got %d", len(fetcher.requests))
	}
	if len(repo.advanceParams) != 1 {
		t.Errorf("expected 1 advance call, got %d", len(repo.advanceParams))
	}
	if repo.advanceParams[0].LastNSU != 10 {
		t.Errorf("expected advance NSU 10, got %d", repo.advanceParams[0].LastNSU)
	}
	assertSingleFinishRun(t, repo, "completed")
}

func TestSyncService_Failure(t *testing.T) {
	repo := &mockSyncRepo{}
	fetcher := &mockFetcher{
		responses: []func() (*adn.DocumentResponse, error){
			func() (*adn.DocumentResponse, error) {
				return nil, errors.New("api error")
			},
		},
	}
	xmlStore := &mockXMLStore{}

	svc := NewSyncService(repo, fetcher, xmlStore, slog.Default())

	company := &nfse.Company{
		ID:      "comp-1",
		CNPJ:    "12345678901234",
		LastNSU: 0,
	}
	credential := &nfse.Credential{
		ID:        "cred-1",
		OwnerCNPJ: "12345678901234",
	}

	err := svc.Sync(context.Background(), company, credential, "exact", nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	assertSingleFinishRun(t, repo, "failed")
}

func TestSyncService_CancellationFinishesOnceAsInterrupted(t *testing.T) {
	repo := &mockSyncRepo{}
	fetcher := &mockFetcher{}
	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())

	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", LastNSU: 0}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.Sync(ctx, company, credential, "exact", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	assertSingleFinishRun(t, repo, "interrupted")
}

func TestSyncService_ProcessingFailureAdvancesCheckpointBeforeFailing(t *testing.T) {
	repo := &mockSyncRepo{}
	repo.applyDocumentErr = func(p nfse.ApplyDocumentParams) error {
		if p.NSU == 2 {
			return errors.New("apply failed")
		}
		return nil
	}
	fetcher := &mockFetcher{
		responses: []func() (*adn.DocumentResponse, error){
			func() (*adn.DocumentResponse, error) {
				return &adn.DocumentResponse{
					UltNSU: 2,
					MaxNSU: 2,
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
						{NSU: 2, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, validDocumentXML)},
					},
				}, nil
			},
		},
	}

	svc := NewSyncService(repo, fetcher, &mockXMLStore{}, slog.Default())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678000100", LastNSU: 0}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678000100"}

	err := svc.Sync(context.Background(), company, credential, "exact", nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	if len(repo.advanceParams) != 1 {
		t.Fatalf("expected exactly 1 checkpoint advance, got %d", len(repo.advanceParams))
	}
	if repo.advanceParams[0].LastNSU != 1 {
		t.Fatalf("expected checkpoint at NSU 1, got %d", repo.advanceParams[0].LastNSU)
	}
	assertSingleFinishRun(t, repo, "failed")
}

func TestSyncService_ProcessesEventFromTipoEventoMetadata(t *testing.T) {
	repo := &mockSyncRepo{}
	fetcher := &mockFetcher{
		responses: []func() (*adn.DocumentResponse, error){
			func() (*adn.DocumentResponse, error) {
				return &adn.DocumentResponse{
					UltNSU: 1,
					MaxNSU: 1,
					Docs: []adn.DocumentEnvelope{
						{
							NSU:           1,
							Schema:        "procNFSe_v1.00.xsd",
							XMLGZipBase64: mustEncodeGzipBase64(t, `<pedCancNFSe><infPedidoCanc><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><cMotivo>Erro emissao</cMotivo><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`),
							EventType:     "CANCELAMENTO",
						},
					},
				}, nil
			},
		},
	}
	xmlStore := &mockXMLStore{}

	svc := NewSyncService(repo, fetcher, xmlStore, slog.Default())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", LastNSU: 0}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact", nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyEventParams) != 1 {
		t.Fatalf("expected 1 event to be applied, got %d", len(repo.applyEventParams))
	}
	if len(repo.applyDocParams) != 0 {
		t.Fatalf("expected no documents to be applied, got %d", len(repo.applyDocParams))
	}
}

func TestSyncService_FallsBackToSchemaWhenMetadataMissing(t *testing.T) {
	repo := &mockSyncRepo{}
	fetcher := &mockFetcher{
		responses: []func() (*adn.DocumentResponse, error){
			func() (*adn.DocumentResponse, error) {
				return &adn.DocumentResponse{
					UltNSU: 2,
					MaxNSU: 2,
					Docs: []adn.DocumentEnvelope{
						{
							NSU:           2,
							Schema:        "procEventoNFSe_v1.00.xsd",
							XMLGZipBase64: mustEncodeGzipBase64(t, `<pedCancNFSe><infPedidoCanc><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><cMotivo>Erro emissao</cMotivo><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`),
						},
					},
				}, nil
			},
		},
	}
	xmlStore := &mockXMLStore{}

	svc := NewSyncService(repo, fetcher, xmlStore, slog.Default())
	company := &nfse.Company{ID: "comp-1", CNPJ: "12345678901234", LastNSU: 0}
	credential := &nfse.Credential{ID: "cred-1", OwnerCNPJ: "12345678901234"}

	if err := svc.Sync(context.Background(), company, credential, "exact", nil); err != nil {
		t.Fatalf("expected sync success, got %v", err)
	}

	if len(repo.applyEventParams) != 1 {
		t.Fatalf("expected 1 event to be applied, got %d", len(repo.applyEventParams))
	}
	if len(repo.applyDocParams) != 0 {
		t.Fatalf("expected no documents to be applied, got %d", len(repo.applyDocParams))
	}
}

func TestNextNSU(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		requestedNSU    int64
		apiUltNSU       int64
		batchSuccessNSU int64
		emptyBatch      bool
		want            int64
	}{
		{name: "empty batch advances manually", requestedNSU: 10, apiUltNSU: 10, emptyBatch: true, want: 11},
		{name: "empty batch uses api ult", requestedNSU: 10, apiUltNSU: 15, emptyBatch: true, want: 15},
		{name: "non empty prefers api ult", apiUltNSU: 20, batchSuccessNSU: 18, want: 20},
		{name: "non empty falls back to processed nsu", apiUltNSU: 18, batchSuccessNSU: 20, want: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextNSU(tt.requestedNSU, tt.apiUltNSU, tt.batchSuccessNSU, tt.emptyBatch); got != tt.want {
				t.Fatalf("nextNSU() = %d, want %d", got, tt.want)
			}
		})
	}
}

func assertSingleFinishRun(t *testing.T, repo *mockSyncRepo, wantStatus nfse.SyncStatus) {
	t.Helper()
	if len(repo.finishRunParams) != 1 {
		t.Fatalf("expected exactly 1 finish call, got %d", len(repo.finishRunParams))
	}
	if repo.finishRunParams[0].Status != wantStatus {
		t.Fatalf("finish status = %s, want %s", repo.finishRunParams[0].Status, wantStatus)
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
