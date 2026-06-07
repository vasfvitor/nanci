package syncservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/nfse"
)

type mockSyncRepo struct {
	startRunParams    []nfse.StartRunParams
	advanceParams     []nfse.AdvanceCheckpointParams
	finishRunParams   []nfse.FinishRunParams
	applyDocParams    []nfse.ApplyDocumentParams
	applyEventParams  []nfse.ApplyEventParams
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
	return nil
}

func (m *mockSyncRepo) ApplyEvent(ctx context.Context, p nfse.ApplyEventParams) error {
	m.applyEventParams = append(m.applyEventParams, p)
	return nil
}

type mockFetcher struct {
	requests []adn.DistributionRequest
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
					Docs:   nil, // Empty batch just to advance NSU
				}, nil
			},
		},
	}
	xmlStore := &mockXMLStore{}
	logger := slog.Default()

	svc := NewSyncService(repo, fetcher, xmlStore, logger)

	company := &nfse.Company{
		ID:       "comp-1",
		CNPJ:     "12345678901234",
		LastNSU:  0,
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
	if len(repo.finishRunParams) != 1 {
		t.Errorf("expected 1 finish call, got %d", len(repo.finishRunParams))
	}
	if repo.finishRunParams[0].Status != "completed" {
		t.Errorf("expected status completed, got %s", repo.finishRunParams[0].Status)
	}
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
		ID:       "comp-1",
		CNPJ:     "12345678901234",
		LastNSU:  0,
	}
	credential := &nfse.Credential{
		ID:        "cred-1",
		OwnerCNPJ: "12345678901234",
	}

	err := svc.Sync(context.Background(), company, credential, "exact", nil)
	if err == nil {
		t.Fatal("expected sync failure, got nil")
	}

	if len(repo.finishRunParams) != 1 {
		t.Errorf("expected 1 finish call, got %d", len(repo.finishRunParams))
	}
	if repo.finishRunParams[0].Status != "failed" {
		t.Errorf("expected status failed, got %s", repo.finishRunParams[0].Status)
	}
}
