package adn

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

func Test_truncateForLog(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int
		want  string
	}{
		{name: "under limit", body: "abc", limit: 10, want: "abc"},
		{name: "at limit", body: "abcde", limit: 5, want: "abcde"},
		{name: "over limit", body: "abcdefgh", limit: 5, want: "abcde... (truncated)"},
		// "ação" is a(1) ç(2) ã(2) o(1) bytes; a cut inside ç or ã backs up.
		{name: "does not split rune", body: "ação", limit: 2, want: "a... (truncated)"},
		{name: "cut lands on rune start", body: "ação", limit: 3, want: "aç... (truncated)"},
		{name: "cut inside second rune", body: "ação", limit: 4, want: "aç... (truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateForLog([]byte(tt.body), tt.limit); got != tt.want {
				t.Errorf("truncateForLog(%q, %d) = %q, want %q", tt.body, tt.limit, got, tt.want)
			}
		})
	}
}

func Test_sanitizeURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "relative path with cnpj", raw: "DFe/10?cnpjConsulta=12345678000195&lote=true", want: "DFe/10?cnpjConsulta=12**********95&lote=true"},
		{name: "absolute url with cnpj", raw: "https://adn.nfse.gov.br/contribuintes/DFe/0?cnpjConsulta=12345678000195", want: "https://adn.nfse.gov.br/contribuintes/DFe/0?cnpjConsulta=12**********95"},
		{name: "no cnpj is untouched", raw: "DFe/10?lote=true", want: "DFe/10?lote=true"},
		{name: "no query is untouched", raw: "DFe/10", want: "DFe/10"},
		{name: "short value fully masked", raw: "x?cnpjConsulta=abc", want: "x?cnpjConsulta=***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeURL(tt.raw); got != tt.want {
				t.Errorf("sanitizeURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func newLoggedClient(t *testing.T, baseURL string, level slog.Level) (*Client, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	client, err := NewClient(ClientConfig{
		BaseURL: baseURL,
		Log:     slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})),
		Retry:   RetryConfig{MaxRetries: 1},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, &buf
}

func TestClient_ErrorLogsAreBoundedAndMasked(t *testing.T) {
	bigBody := strings.Repeat("é", maxErrorLogBodyBytes) // 2 bytes each: twice the log cap
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(bigBody))
	}))
	defer server.Close()

	client, logs := newLoggedClient(t, server.URL, slog.LevelDebug)
	err := client.RawGet(context.Background(), "DFe/1?cnpjConsulta=12345678000195", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	out := logs.String()
	if strings.Contains(out, "12345678000195") {
		t.Errorf("log leaks the consulted CNPJ:\n%s", out)
	}
	if !strings.Contains(out, "cnpjConsulta=12**********95") {
		t.Errorf("expected masked cnpjConsulta in log:\n%s", out)
	}
	if !strings.Contains(out, "(truncated)") {
		t.Errorf("expected truncated error body in log:\n%s", out)
	}
	if strings.Contains(out, "ADN API Error Response Body") {
		t.Errorf("full body must not be logged below trace level:\n%s", out)
	}
	if len(out) > 2*maxErrorLogBodyBytes+1024 {
		t.Errorf("log record too large (%d bytes) for a %d byte cap", len(out), maxErrorLogBodyBytes)
	}

	msg := err.Error()
	if strings.Contains(msg, "12345678000195") {
		t.Errorf("error message leaks the consulted CNPJ: %s", msg)
	}
	if !strings.HasSuffix(msg, "(truncated)") {
		t.Errorf("error message should carry a truncated body, got %d bytes", len(msg))
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.Body) != len(bigBody) {
		t.Errorf("APIError.Body should keep the full body")
	}
}

func TestClient_ResponseBodyOnlyAtTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"secret":"base64xml"}`))
	}))
	defer server.Close()

	client, debugLogs := newLoggedClient(t, server.URL, slog.LevelDebug)
	if err := client.RawGet(context.Background(), "DFe/1", nil); err != nil {
		t.Fatalf("RawGet: %v", err)
	}
	if strings.Contains(debugLogs.String(), "base64xml") {
		t.Errorf("response body must not be logged at debug:\n%s", debugLogs.String())
	}
	if !strings.Contains(debugLogs.String(), "body_bytes=") {
		t.Errorf("expected body size in debug log:\n%s", debugLogs.String())
	}

	client, traceLogs := newLoggedClient(t, server.URL, logger.LevelTrace)
	if err := client.RawGet(context.Background(), "DFe/1", nil); err != nil {
		t.Fatalf("RawGet: %v", err)
	}
	if !strings.Contains(traceLogs.String(), "base64xml") {
		t.Errorf("expected response body at trace level:\n%s", traceLogs.String())
	}
}
