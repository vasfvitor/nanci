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

	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

// testChave is a 50-digit NFS-e access key; maskedChave is how the log shows it.
const (
	testChave   = "35503082112223330001811000000000000012345123456789"
	maskedChave = "35**********************************************89"
)

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
		{name: "relative eventos path", raw: "NFSe/" + testChave + "/Eventos", want: "NFSe/" + maskedChave + "/Eventos"},
		{name: "absolute eventos path", raw: "https://adn.nfse.gov.br/contribuintes/NFSe/" + testChave + "/Eventos", want: "https://adn.nfse.gov.br/contribuintes/NFSe/" + maskedChave + "/Eventos"},
		{name: "nfse without key is untouched", raw: "NFSe/", want: "NFSe/"},
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
	bigBody := strings.Repeat("é", httpclient.MaxErrorLogBodyBytes) // 2 bytes each: twice the log cap
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
	if len(out) > 2*httpclient.MaxErrorLogBodyBytes+1024 {
		t.Errorf("log record too large (%d bytes) for a %d byte cap", len(out), httpclient.MaxErrorLogBodyBytes)
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

func TestClient_ErrorBodiesHideIdentifiers(t *testing.T) {
	const identifiers = "<CNPJ>11222333000181</CNPJ><chNFSe>" + testChave + "</chNFSe>"
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"400 with XML", http.StatusBadRequest, "<Erro>" + identifiers + "</Erro>"},
		{"404 outside the ADN envelope", http.StatusNotFound, "<html>" + identifiers + "</html>"},
		// The hand-written ErrorContext in notFound logs this one.
		{"404 inside the ADN envelope", http.StatusNotFound, `{"Erros":[{"Codigo":"E9999","Descricao":"` + identifiers + `"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, logs := newLoggedClient(t, server.URL, logger.LevelTrace)
			err := client.RawGet(context.Background(), "NFSe/"+testChave+"/Eventos", nil)
			if err == nil {
				t.Fatal("expected error")
			}

			out, msg := logs.String(), err.Error()
			for _, clear := range []string{"11222333000181", testChave} {
				if strings.Contains(out, clear) {
					t.Errorf("log leaks %q:\n%s", clear, out)
				}
				if strings.Contains(msg, clear) {
					t.Errorf("error message leaks %q: %s", clear, msg)
				}
			}
			if !strings.Contains(msg, "<CNPJ>11**********81</CNPJ>") {
				t.Errorf("expected the masked CNPJ in the error message: %s", msg)
			}
			if !strings.Contains(out, "NFSe/"+maskedChave+"/Eventos") {
				t.Errorf("expected the masked access key in the logged URL:\n%s", out)
			}
		})
	}
}

func TestClient_TraceBodyHidesIdentifiers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<NFSe><CNPJ>11222333000181</CNPJ><chNFSe>" + testChave + "</chNFSe><xNome>PRESTADORA FICTICIA</xNome></NFSe>"))
	}))
	defer server.Close()

	client, logs := newLoggedClient(t, server.URL, logger.LevelTrace)
	if err := client.RawGet(context.Background(), "DFe/1", nil); err != nil {
		t.Fatalf("RawGet: %v", err)
	}

	out := logs.String()
	if !strings.Contains(out, "ADN API Response Body") {
		t.Fatalf("expected the response body at trace level:\n%s", out)
	}
	for _, clear := range []string{"11222333000181", testChave, "PRESTADORA FICTICIA"} {
		if strings.Contains(out, clear) {
			t.Errorf("log leaks %q:\n%s", clear, out)
		}
	}
	if !strings.Contains(out, "<CNPJ>11**********81</CNPJ>") {
		t.Errorf("expected the masked CNPJ in the log:\n%s", out)
	}
}
