package sefaz

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
)

// CheckTLS opens a TLS connection to the NF-e distribution host with the
// same configuration as the SOAP calls and closes it right after the
// handshake. It sends no request, so it does not count against the SEFAZ
// hourly limit.
func (c *Client) CheckTLS(ctx context.Context) error {
	return c.checkTLS(ctx, c.endpoints.Distribuicao)
}

// CheckTLSCTe does the same as CheckTLS with the CT-e distribution host.
func (c *Client) CheckTLSCTe(ctx context.Context) error {
	return c.checkTLS(ctx, c.endpoints.DistribuicaoCTe)
}

func (c *Client) checkTLS(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid distribution URL: %w", err)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("invalid distribution URL %q: no host", rawURL)
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	dialer := &tls.Dialer{Config: c.tlsConfig.Clone()}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(u.Hostname(), port))
	if err != nil {
		return fmt.Errorf("TLS handshake with %s: %w", u.Hostname(), err)
	}
	return conn.Close()
}
