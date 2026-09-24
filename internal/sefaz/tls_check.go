package sefaz

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
)

// CheckTLS opens a TLS connection to the distribution host with the same
// configuration as the SOAP calls and closes it right after the handshake.
// It sends no request, so it does not count against the SEFAZ hourly limit.
func (c *Client) CheckTLS(ctx context.Context) error {
	u, err := url.Parse(c.endpoints.Distribuicao)
	if err != nil {
		return fmt.Errorf("invalid distribution URL: %w", err)
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
