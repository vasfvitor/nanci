package adn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
	}{
		{
			name: "valid minimal config",
			cfg: ClientConfig{
				BaseURL: BaseURLProduction,
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			cfg: ClientConfig{
				BaseURL: "",
			},
			wantErr: true,
		},
		{
			name: "negative retries",
			cfg: ClientConfig{
				BaseURL: BaseURLProduction,
				Retry: RetryConfig{
					MaxRetries: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid base URL",
			cfg: ClientConfig{
				BaseURL: ":\x00invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_RawGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"test": "ok"}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var dest map[string]string
	err = client.RawGet(context.Background(), "/some-path", &dest)
	if err != nil {
		t.Errorf("RawGet() error = %v", err)
	}
	if dest["test"] != "ok" {
		t.Errorf("expected dest['test'] == 'ok', got %v", dest["test"])
	}
}

func TestClient_RawGet_Retries(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests) // retryable
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL: server.URL,
		Retry: RetryConfig{
			MaxRetries: 3,
			Initial:    1 * time.Millisecond,
			MaxDelay:   5 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var dest map[string]bool
	err = client.RawGet(context.Background(), "/retry-path", &dest)
	if err != nil {
		t.Errorf("RawGet() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if !dest["success"] {
		t.Errorf("expected dest['success'] == true")
	}
}

func TestClient_RawGet_NoDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{
			"StatusProcessamento": "NENHUM_DOCUMENTO_LOCALIZADO",
			"LoteDFe": [],
			"Erros": [{"Codigo": "E2220"}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var dest map[string]any
	err = client.RawGet(context.Background(), "/no-docs", &dest)
	if !errors.Is(err, ErrNoDocumentsLocated) {
		t.Errorf("expected ErrNoDocumentsLocated, got %v", err)
	}
}

func Test_isNoDocumentsResponse(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "valid NENHUM_DOCUMENTO_LOCALIZADO",
			body: `{"StatusProcessamento":"NENHUM_DOCUMENTO_LOCALIZADO","Erros":[{"Codigo":"E2220"}]}`,
			want: true,
		},
		{
			name: "wrong status",
			body: `{"StatusProcessamento":"PROCESSADO","Erros":[{"Codigo":"E2220"}]}`,
			want: false,
		},
		{
			name: "wrong error code",
			body: `{"StatusProcessamento":"NENHUM_DOCUMENTO_LOCALIZADO","Erros":[{"Codigo":"E0000"}]}`,
			want: false,
		},
		{
			name: "invalid json",
			body: `invalid json`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNoDocumentsResponse([]byte(tt.body)); got != tt.want {
				t.Errorf("isNoDocumentsResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_classifyUnexpected404Body(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "empty body",
			body: "   ",
			want: "unexpected 404 from ADN route: empty response body",
		},
		{
			name: "html response",
			body: "<html><body>Not Found</body></html>",
			want: "unexpected 404 from ADN route: non-ADN HTML response: <html><body>Not Found</body></html>",
		},
		{
			name: "valid adn error response format",
			body: `{"StatusProcessamento":"ERRO"}`,
			want: "",
		},
		{
			name: "valid adn ok response format",
			body: `{"LoteDFe":[]}`,
			want: "",
		},
		{
			name: "unexpected json payload",
			body: `{"message":"not found"}`,
			want: `unexpected 404 from ADN route: payload does not match ADN envelope: {"message":"not found"}`,
		},
		{
			name: "non json",
			body: "plain text error",
			want: "unexpected 404 from ADN route: non-ADN payload: plain text error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyUnexpected404Body([]byte(tt.body)); got != tt.want {
				t.Errorf("classifyUnexpected404Body() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		maxDelay time.Duration
		want     time.Duration
	}{
		{"empty string", "", 10 * time.Second, 0},
		{"valid seconds", "5", 10 * time.Second, 5 * time.Second},
		{"valid seconds with spaces", " 3 ", 10 * time.Second, 3 * time.Second},
		{"exceeds max delay", "20", 10 * time.Second, 10 * time.Second},
		{"invalid format", "abc", 10 * time.Second, 0},
		{"negative number", "-5", 10 * time.Second, 0},
		{"no max delay limit", "15", 0, 15 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseRetryAfter(tt.raw, tt.maxDelay); got != tt.want {
				t.Errorf("parseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isRetryableStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{"200 OK", 200, false},
		{"400 Bad Request", 400, false},
		{"404 Not Found", 404, false},
		{"408 Request Timeout", 408, true},
		{"425 Too Early", 425, true},
		{"429 Too Many Requests", 429, true},
		{"500 Internal Server Error", 500, true},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
		{"504 Gateway Timeout", 504, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableStatus(tt.status); got != tt.want {
				t.Errorf("isRetryableStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
