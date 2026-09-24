package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
)

func TestNewTransport(t *testing.T) {
	cert := &tls.Certificate{}
	roots := x509.NewCertPool()

	transport := NewTransport(cert, roots)
	cfg := transport.TLSClientConfig
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want TLS 1.2", cfg.MinVersion)
	}
	if cfg.Renegotiation != tls.RenegotiateFreelyAsClient {
		t.Errorf("Renegotiation = %v, want RenegotiateFreelyAsClient", cfg.Renegotiation)
	}
	if transport.ForceAttemptHTTP2 {
		t.Error("HTTP/2 must be off")
	}
	if cfg.RootCAs != roots {
		t.Error("RootCAs not set")
	}
	got, err := cfg.GetClientCertificate(&tls.CertificateRequestInfo{})
	if err != nil || got != cert {
		t.Errorf("GetClientCertificate = %v, %v; want the configured certificate", got, err)
	}
}

func TestNewTransport_NoCertificate(t *testing.T) {
	transport := NewTransport(nil, nil)
	if transport.TLSClientConfig.GetClientCertificate != nil {
		t.Error("GetClientCertificate must be nil without a certificate")
	}
	if transport.TLSClientConfig.RootCAs != nil {
		t.Error("RootCAs must be nil to use the platform roots")
	}
}
