package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

// NewTransport returns a clone of http.DefaultTransport (keeping its proxy
// and dial settings) configured for the government web services: TLS 1.2 or
// newer, renegotiation allowed, HTTP/2 off, and cert presented when the
// server asks for a client certificate. A nil roots pool uses the platform
// roots.
func NewTransport(cert *tls.Certificate, roots *x509.CertPool) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	tlsConfig := &tls.Config{
		MinVersion:    tls.VersionTLS12,
		Renegotiation: tls.RenegotiateFreelyAsClient,
		RootCAs:       roots,
	}
	if cert != nil {
		tlsConfig.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return cert, nil
		}
	}

	transport.TLSClientConfig = tlsConfig
	transport.ForceAttemptHTTP2 = false
	return transport
}
