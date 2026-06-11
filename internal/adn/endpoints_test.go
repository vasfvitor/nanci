package adn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		if got := r.URL.Query().Get("cnpjConsulta"); got != "12345678901234" {
			t.Errorf("expected cnpjConsulta=12345678901234, got %s", got)
		}
		if got := r.URL.Query().Get("lote"); got != "" {
			t.Errorf("expected lote to be omitted by default, got %q", got)
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
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.FetchDocuments(ctx, DistributionRequest{
		LastNSU:          100,
		ConsultationCNPJ: "12345678901234",
	})
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

func TestClient_FetchDocuments_SerializesOptionalLote(t *testing.T) {
	tests := []struct {
		name     string
		lote     *bool
		expected string
	}{
		{name: "omitted by default", lote: nil, expected: ""},
		{name: "explicit true", lote: boolPtr(true), expected: "true"},
		{name: "explicit false", lote: boolPtr(false), expected: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/DFe/0", func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("lote"); got != tt.expected {
					t.Fatalf("expected lote=%q, got %q", tt.expected, got)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(DocumentResponse{})
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			client := newTestClient(t, server.URL)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0, Lote: tt.lote})
			if err != nil {
				t.Fatalf("FetchDocuments failed: %v", err)
			}
		})
	}
}

func TestClient_Retries(t *testing.T) {
	var requestCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DocumentResponse{
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
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

func TestClient_RetryAfterHeaderOverridesBackoff(t *testing.T) {
	var requestCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DocumentResponse{UltNSU: 1, MaxNSU: 1})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentProduction,
		BaseURLOverride: server.URL,
		Retry: RetryConfig{
			MaxRetries: 2,
			Initial:    1 * time.Millisecond,
			MaxDelay:   1500 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	_, err = client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
	if err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}
	elapsed := time.Since(start)

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected Retry-After delay to be respected, got %s", elapsed)
	}
}

func TestClient_RetryAfterInvalidFallsBackToBackoff(t *testing.T) {
	var requestCount int
	mux := http.NewServeMux()
	mux.HandleFunc("/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Retry-After", "invalid")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DocumentResponse{UltNSU: 1, MaxNSU: 1})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentProduction,
		BaseURLOverride: server.URL,
		Retry: RetryConfig{
			MaxRetries: 2,
			Initial:    1 * time.Millisecond,
			MaxDelay:   5 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err = client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
	if err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}
	elapsed := time.Since(start)

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("expected fallback backoff to stay short, got %s", elapsed)
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

	client := newRestrictedTestClient(t, server.URL+"/contribuintes")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0}); err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}
}

func TestClient_FetchDocuments_NoDocumentsResponseIsEmptyBatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/contribuintes/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"StatusProcessamento": "NENHUM_DOCUMENTO_LOCALIZADO",
			"LoteDFe":             []any{},
			"Alertas":             []any{},
			"Erros": []map[string]any{
				{"Codigo": "E2220", "Descricao": "Nenhum documento localizado"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newRestrictedTestClient(t, server.URL+"/contribuintes")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
	if err != nil {
		t.Fatalf("FetchDocuments failed: %v", err)
	}
	if resp.UltNSU != 0 {
		t.Errorf("expected ultNSU 0, got %d", resp.UltNSU)
	}
	if resp.MaxNSU != 0 {
		t.Errorf("expected maxNSU 0, got %d", resp.MaxNSU)
	}
	if len(resp.Docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(resp.Docs))
	}
}

func TestClient_FetchDocuments_Unexpected404HTMLIsInfrastructureError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/contribuintes/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html><body>proxy miss</body></html>"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newRestrictedTestClient(t, server.URL+"/contribuintes")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
	if err == nil {
		t.Fatal("expected FetchDocuments to fail on HTML 404")
	}
	if !strings.Contains(err.Error(), "non-ADN HTML response") {
		t.Fatalf("expected explicit HTML infrastructure error, got %v", err)
	}
}

func TestClient_FetchDocuments_Unexpected404EmptyBodyIsInfrastructureError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/contribuintes/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newRestrictedTestClient(t, server.URL+"/contribuintes")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
	if err == nil {
		t.Fatal("expected FetchDocuments to fail on empty 404 body")
	}
	if !strings.Contains(err.Error(), "empty response body") {
		t.Fatalf("expected explicit empty-body infrastructure error, got %v", err)
	}
}

func TestClient_FetchDocuments_Unexpected404JSONWithoutADNShapeIsInfrastructureError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/contribuintes/DFe/0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "cdn miss"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newRestrictedTestClient(t, server.URL+"/contribuintes")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.FetchDocuments(ctx, DistributionRequest{LastNSU: 0})
	if err == nil {
		t.Fatal("expected FetchDocuments to fail on malformed ADN 404 payload")
	}
	if !strings.Contains(err.Error(), "does not match ADN envelope") {
		t.Fatalf("expected explicit malformed-envelope error, got %v", err)
	}
}

func TestDocumentResponse_UnmarshalOfficialEnvelope(t *testing.T) {
	payload := []byte(`{
		"UltNSU": 15,
		"MaxNSU": 20,
		"LoteDFe": [
			{
				"NSU": 16,
				"Schema": "procNFSe_v1.00.xsd",
				"ArquivoXml": "payload_1",
				"TipoDocumento": "NFSE",
				"TipoEvento": ""
			}
		]
	}`)

	var resp DocumentResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if resp.UltNSU != 15 || resp.MaxNSU != 20 {
		t.Fatalf("unexpected NSU range: %+v", resp)
	}
	if len(resp.Docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(resp.Docs))
	}
	if got := resp.Docs[0].PayloadBase64(); got != "payload_1" {
		t.Fatalf("expected ArquivoXml payload, got %q", got)
	}
	if resp.Docs[0].DocumentType != "NFSE" {
		t.Fatalf("expected TipoDocumento to be preserved, got %q", resp.Docs[0].DocumentType)
	}
}

func TestDocumentResponse_UnmarshalLegacyEnvelopeFallback(t *testing.T) {
	payload := []byte(`{
		"ultNSU": 10,
		"maxNSU": 10,
		"docFisc": [
			{
				"nsu": 10,
				"schema": "procNFSe_v1.00.xsd",
				"nfseXmlGZipB64": "payload_legacy"
			}
		]
	}`)

	var resp DocumentResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp.Docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(resp.Docs))
	}
	if got := resp.Docs[0].PayloadBase64(); got != "payload_legacy" {
		t.Fatalf("expected legacy payload, got %q", got)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentProduction,
		BaseURLOverride: baseURL,
		Retry: RetryConfig{
			MaxRetries: 0,
			Initial:    1 * time.Millisecond,
			MaxDelay:   5 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func newRestrictedTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(ClientConfig{
		Environment:     nfse.EnvironmentRestricted,
		BaseURLOverride: baseURL,
		Retry: RetryConfig{
			MaxRetries: 0,
			Initial:    1 * time.Millisecond,
			MaxDelay:   5 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func boolPtr(v bool) *bool {
	return &v
}
