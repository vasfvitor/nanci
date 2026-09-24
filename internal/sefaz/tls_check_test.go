package sefaz

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestCheckTLS(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()

	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	newClient := func(roots *x509.CertPool) *Client {
		client, err := NewClient(ClientConfig{
			Environment: nfse.EnvironmentProduction,
			Certificate: &tls.Certificate{},
			RootCAs:     roots,
			Endpoints:   &Endpoints{Distribuicao: server.URL + "/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx"},
		})
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		return client
	}

	if err := newClient(roots).CheckTLS(context.Background()); err != nil {
		t.Fatalf("CheckTLS with the server root: %v", err)
	}
	if err := newClient(nil).CheckTLS(context.Background()); err == nil {
		t.Error("CheckTLS must fail when the server certificate is not trusted")
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("CheckTLS sent %d HTTP requests, want 0", n)
	}
}
