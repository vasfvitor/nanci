package adn

import (
	"crypto/tls"
	"net/http"
	"time"
)

// DefaultTimeout is the default timeout for API requests.
const DefaultTimeout = 30 * time.Second

// NewTLSConfig creates a tls.Config from a loaded certificate.
// The ADN API requires mTLS (Mutual TLS).
func NewTLSConfig(cert *tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		// Renegotiation is often needed for some government APIs
		Renegotiation: tls.RenegotiateOnceAsClient,
		// InsecureSkipVerify might be needed if the API uses an internal CA that we don't have,
		// but ideally we should trust the official ICP-Brasil chain.
		// For now, let's keep it secure.
		MinVersion: tls.VersionTLS12,
	}
}

// NewHTTPClient creates an HTTP client configured with mTLS for the given certificate.
func NewHTTPClient(cert *tls.Certificate) *http.Client {
	tlsConfig := NewTLSConfig(cert)

	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   DefaultTimeout,
	}
}
