package sefaz

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const soapFault12 = `<?xml version="1.0" encoding="utf-8"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
	`<soap:Fault><soap:Code><soap:Value>soap:Receiver</soap:Value></soap:Code>` +
	`<soap:Reason><soap:Text xml:lang="pt-BR">Server was unable to process request.</soap:Text></soap:Reason>` +
	`</soap:Fault></soap:Body></soap:Envelope>`

func TestNewClient_Validation(t *testing.T) {
	if _, err := NewClient(ClientConfig{Environment: nfse.EnvironmentProduction}); err == nil {
		t.Error("expected error without a certificate")
	}
	if _, err := NewClient(ClientConfig{Environment: "local", Certificate: &tls.Certificate{}}); !errors.Is(err, nfse.ErrInvalidEnum) {
		t.Errorf("invalid environment: err = %v, want ErrInvalidEnum", err)
	}
}

func TestNewClient_DefaultsFromEnvironment(t *testing.T) {
	client, err := NewClient(ClientConfig{Environment: nfse.EnvironmentRestricted, Certificate: &tls.Certificate{}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.tpAmb != TpAmbHomologacao {
		t.Errorf("tpAmb = %q, want %q", client.tpAmb, TpAmbHomologacao)
	}
	if client.endpoints.Distribuicao != DistribuicaoHomologacao || client.endpoints.DistribuicaoCTe != DistribuicaoCTeHomologacao ||
		client.endpoints.RecepcaoEvento != RecepcaoEventoHomologacao {
		t.Errorf("endpoints = %+v", client.endpoints)
	}
	if client.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", client.timeout, DefaultTimeout)
	}
}

// The Ambiente Nacional asks for the client certificate through a TLS
// renegotiation, which Go refuses by default.
func TestNewClient_TransportAllowsRenegotiation(t *testing.T) {
	client, err := NewClient(ClientConfig{Environment: nfse.EnvironmentProduction, Certificate: &tls.Certificate{}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.tlsConfig.Renegotiation != tls.RenegotiateFreelyAsClient {
		t.Errorf("Renegotiation = %v, want RenegotiateFreelyAsClient", client.tlsConfig.Renegotiation)
	}
	if client.tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want TLS 1.2", client.tlsConfig.MinVersion)
	}
	if client.tlsConfig.GetClientCertificate == nil {
		t.Error("the client certificate must be offered when the server asks")
	}
}

func TestPost_SOAPFaultIsNotRetried(t *testing.T) {
	client, fake := newFakeClient(t, http.StatusInternalServerError, soapFault12, ClientConfig{})

	_, err := client.DistNSU(context.Background(), testCNPJ, 35, 0)
	var fault *FaultError
	if !errors.As(err, &fault) {
		t.Fatalf("err = %v, want *FaultError", err)
	}
	if fault.StatusCode != http.StatusInternalServerError || fault.Code != "soap:Receiver" || fault.Reason != "Server was unable to process request." {
		t.Errorf("fault = %+v", fault)
	}
	if n := len(fake.captured()); n != 1 {
		t.Errorf("sent %d requests, want 1", n)
	}
}

// httpclient logs an accepted HTTP 500 at Debug only, so a SOAP Fault must
// be logged at Error level by the SEFAZ client, with the body redacted.
func TestPost_SOAPFaultIsLoggedAsError(t *testing.T) {
	faultWithCNPJ := strings.Replace(soapFault12, "Server was unable to process request.",
		"Server was unable to process request.</soap:Text><soap:Text><CNPJ>"+testCNPJ+"</CNPJ>", 1)

	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError}))
	client, _ := newFakeClient(t, http.StatusInternalServerError, faultWithCNPJ, ClientConfig{Log: log})

	_, err := client.DistNSU(context.Background(), testCNPJ, 35, 0)
	var fault *FaultError
	if !errors.As(err, &fault) {
		t.Fatalf("err = %v, want *FaultError", err)
	}

	out := logs.String()
	for _, want := range []string{"level=ERROR", "SEFAZ Error Response", "status=500", "Server was unable to process request."} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, testCNPJ) {
		t.Errorf("log leaks the CNPJ:\n%s", out)
	}
}

func TestPost_ServerErrorsAreNotRetried(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"503", http.StatusServiceUnavailable, "Service Unavailable"},
		{"500 without a fault", http.StatusInternalServerError, "<html>error</html>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fake := newFakeClient(t, tt.status, tt.body, ClientConfig{})

			_, err := client.DistNSU(context.Background(), testCNPJ, 35, 0)
			var statusErr *httpclient.StatusError
			if !errors.As(err, &statusErr) || statusErr.StatusCode != tt.status {
				t.Fatalf("err = %v, want StatusError %d", err, tt.status)
			}
			if n := len(fake.captured()); n != 1 {
				t.Errorf("sent %d requests, want 1", n)
			}
		})
	}
}

func TestPost_TransportErrorRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		// Drop the connection without an HTTP response.
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment: nfse.EnvironmentProduction,
		Certificate: &tls.Certificate{},
		Endpoints:   &Endpoints{Distribuicao: server.URL, RecepcaoEvento: server.URL},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for _, tt := range []struct {
		retries  int
		attempts int32
	}{{0, 1}, {1, 2}} {
		attempts.Store(0)
		_, err := client.post(context.Background(), server.URL, "urn:test", []byte("<x></x>"), tt.retries)
		if !isTransportError(err) {
			t.Fatalf("retries %d: err = %v, want a transport error", tt.retries, err)
		}
		if got := attempts.Load(); got != tt.attempts {
			t.Errorf("retries %d: %d attempts, want %d", tt.retries, got, tt.attempts)
		}
	}
}

func TestPost_OversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		chunk := bytes.Repeat([]byte("A"), 1<<20)
		for written := 0; written <= MaxResponseBytes; written += len(chunk) {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Environment: nfse.EnvironmentProduction,
		Certificate: &tls.Certificate{},
		Endpoints:   &Endpoints{Distribuicao: server.URL},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.DistNSU(context.Background(), testCNPJ, 35, 0)
	if !errors.Is(err, httpclient.ErrResponseTooLarge) {
		t.Fatalf("err = %v, want ErrResponseTooLarge", err)
	}
}

func TestClient_LogsHideIdentifiers(t *testing.T) {
	const chave = "35260911222333000181550010000012341123456787"
	// The answer carries identifiers in clear, as event answers do.
	response := retDistResponse(retDist(CStatNenhumDocumento, "Nenhum documento localizado", "000000000000001", "000000000000001",
		"<CNPJ>11222333000181</CNPJ><chNFe>"+chave+"</chNFe><xNome>DISTRIBUIDORA FICTICIA</xNome>"))

	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: logger.LevelTrace}))
	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{Log: log})

	if _, err := client.ConsChNFe(context.Background(), testCNPJ, 35, chave); err != nil {
		t.Fatalf("ConsChNFe: %v", err)
	}

	out := logs.String()
	if !strings.Contains(out, "SEFAZ Response Body") {
		t.Fatalf("expected the response body at trace level:\n%s", out)
	}
	for _, clear := range []string{testCNPJ, "11222333000181", chave, "DISTRIBUIDORA FICTICIA"} {
		if strings.Contains(out, clear) {
			t.Errorf("log leaks %q:\n%s", clear, out)
		}
	}
	if !strings.Contains(out, "<CNPJ>11**********81</CNPJ>") {
		t.Errorf("expected the masked CNPJ in the log:\n%s", out)
	}
	if strings.Contains(out, "distDFeInt") {
		t.Errorf("the request body must never be logged:\n%s", out)
	}
}

func TestFindElements(t *testing.T) {
	body := []byte(`<a xmlns:p="urn:p"><p:item n="1"><item>nested</item></p:item><other></other><item n="2"></item></a>`)
	found, err := findElements(body, "item")
	if err != nil {
		t.Fatalf("findElements: %v", err)
	}
	want := []string{`<p:item n="1"><item>nested</item></p:item>`, `<item n="2"></item>`}
	if len(found) != len(want) {
		t.Fatalf("found %d elements, want %d", len(found), len(want))
	}
	for i := range want {
		if string(found[i]) != want[i] {
			t.Errorf("element %d = %s, want %s", i, found[i], want[i])
		}
	}

	var v struct{}
	if err := decodeElement(body, "missing", &v); err == nil {
		t.Error("expected error for a missing element")
	}
	if _, err := findElements([]byte("<a><b></a>"), "b"); err == nil {
		t.Error("expected error for malformed XML")
	}
}
