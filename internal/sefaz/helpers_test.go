package sefaz

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// testCNPJ is the owner of the mock certificate in foundation/cert/testdata.
const testCNPJ = "70860312000150"

type capturedRequest struct {
	Path   string
	Header http.Header
	Body   string
}

// fakeSEFAZ answers every request with the same status and body and records
// what it received.
type fakeSEFAZ struct {
	status int
	body   string

	mu       sync.Mutex
	requests []capturedRequest
}

func (f *fakeSEFAZ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.requests = append(f.requests, capturedRequest{Path: r.URL.Path, Header: r.Header.Clone(), Body: string(body)})
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.WriteHeader(f.status)
	_, _ = io.WriteString(w, f.body)
}

func (f *fakeSEFAZ) captured() []capturedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]capturedRequest(nil), f.requests...)
}

// newFakeClient starts a fakeSEFAZ and a Client whose endpoints all point to
// it. cfg may set Environment and Log; the rest is filled in.
func newFakeClient(t *testing.T, status int, body string, cfg ClientConfig) (*Client, *fakeSEFAZ) {
	t.Helper()
	fake := &fakeSEFAZ{status: status, body: body}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	if cfg.Environment == "" {
		cfg.Environment = nfse.EnvironmentProduction
	}
	if cfg.Certificate == nil {
		cfg.Certificate = &tls.Certificate{}
	}
	cfg.Endpoints = &Endpoints{
		Distribuicao:    server.URL + "/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx",
		DistribuicaoCTe: server.URL + "/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx",
		RecepcaoEvento:  server.URL + "/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx",
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, fake
}

// mustDocZip gzips and base64-encodes a fixture the way SEFAZ ships docZips.
func mustDocZip(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		t.Fatalf("gzip %s: %v", path, err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip %s: %v", path, err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
