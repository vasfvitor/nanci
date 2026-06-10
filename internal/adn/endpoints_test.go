package adn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestClient_FetchDocuments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/DFe/100", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		cnpjQuery := r.URL.Query().Get("cnpjConsulta")
		if cnpjQuery != "12345678901234" {
			t.Errorf("expected cnpjConsulta=12345678901234, got %s", cnpjQuery)
		}

		resp := DocumentResponse{
			UltNSU: 105,
			MaxNSU: 200,
			Docs: []DocumentEnvelope{
				{NSU: 101, Schema: "schema_1", XMLGZipBase64: "base64_1"},
				{NSU: 102, Schema: "schema_1", XMLGZipBase64: "base64_2"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentProduction,
		BaseURLOverride: server.URL,
		Retry: RetryConfig{
			MaxRetries: 0,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := DistributionRequest{
		LastNSU:          100,
		ConsultationCNPJ: "12345678901234",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.FetchDocuments(ctx, req)
	if err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}

	if resp.UltNSU != 105 {
		t.Errorf("expected ultNSU 105, got %d", resp.UltNSU)
	}

	if resp.MaxNSU != 200 {
		t.Errorf("expected maxNSU 200, got %d", resp.MaxNSU)
	}

	if len(resp.Docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(resp.Docs))
	}

	if resp.Docs[0].NSU != 101 {
		t.Errorf("expected doc 0 NSU 101, got %d", resp.Docs[0].NSU)
	}
}

func TestClient_Retries(t *testing.T) {
	var requestCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests) // 429 Retryable
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DocumentResponse{
			UltNSU: 10,
			MaxNSU: 10,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentProduction,
		BaseURLOverride: server.URL,
		Retry: RetryConfig{
			MaxRetries: 3,
			Initial:    1 * time.Millisecond,
			MaxDelay:   5 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := DistributionRequest{LastNSU: 0}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.FetchDocuments(ctx, req)
	if err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}

	if requestCount != 3 {
		t.Errorf("expected 3 requests due to retries, got %d", requestCount)
	}
	if resp.UltNSU != 10 {
		t.Errorf("expected ultNSU 10, got %d", resp.UltNSU)
	}
}

func TestClient_FetchDocuments_WithContribuintesBasePath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/contribuintes/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DocumentResponse{
			UltNSU: 0,
			MaxNSU: 0,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentRestricted,
		BaseURLOverride: server.URL + "/contribuintes",
		Retry: RetryConfig{
			MaxRetries: 0,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0}); err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}
}
