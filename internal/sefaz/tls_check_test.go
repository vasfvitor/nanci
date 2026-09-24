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

	// Each case points only its own endpoint at the server, so a check that
	// dials the wrong host fails on the empty URL.
	tests := []struct {
		name      string
		endpoints Endpoints
		check     func(*Client, context.Context) error
	}{
		{
			name:      "NF-e",
			endpoints: Endpoints{Distribuicao: server.URL + "/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx"},
			check:     (*Client).CheckTLS,
		},
		{
			name:      "CT-e",
			endpoints: Endpoints{DistribuicaoCTe: server.URL + "/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx"},
			check:     (*Client).CheckTLSCTe,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newClient := func(roots *x509.CertPool) *Client {
				client, err := NewClient(ClientConfig{
					Environment: nfse.EnvironmentProduction,
					Certificate: &tls.Certificate{},
					RootCAs:     roots,
					Endpoints:   &tt.endpoints,
				})
				if err != nil {
					t.Fatalf("NewClient: %v", err)
				}
				return client
			}

			if err := tt.check(newClient(roots), context.Background()); err != nil {
				t.Fatalf("check with the server root: %v", err)
			}
			if err := tt.check(newClient(nil), context.Background()); err == nil {
				t.Error("check must fail when the server certificate is not trusted")
			}
		})
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("CheckTLS sent %d HTTP requests, want 0", n)
	}
}

func TestCheckTLS_EmptyURL(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Environment: nfse.EnvironmentProduction,
		Certificate: &tls.Certificate{},
		Endpoints:   &Endpoints{},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.CheckTLS(context.Background()); err == nil {
		t.Error("CheckTLS must fail without a distribution URL")
	}
	if err := client.CheckTLSCTe(context.Background()); err == nil {
		t.Error("CheckTLSCTe must fail without a CT-e distribution URL")
	}
}
