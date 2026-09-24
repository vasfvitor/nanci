// Portions adapted from github.com/mschunke/gonfe (MIT License). See third_party/gonfe/LICENSE.

package sefaz

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
	"github.com/vasfvitor/nanci/internal/foundation/redact"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const (
	// DefaultTimeout bounds each HTTP attempt when ClientConfig.Timeout is zero.
	DefaultTimeout = 60 * time.Second
	// MaxResponseBytes caps a response body: 50 procNFe docZips can take
	// several megabytes.
	MaxResponseBytes = 32 * 1024 * 1024 // 32 MiB

	nfeNamespace  = "http://www.portalfiscal.inf.br/nfe"
	soapNamespace = "http://www.w3.org/2003/05/soap-envelope"
)

// FaultError is a SOAP Fault returned with HTTP 500.
type FaultError struct {
	StatusCode int
	Code       string
	Reason     string
}

func (e *FaultError) Error() string {
	return fmt.Sprintf("SEFAZ SOAP fault (status %d): %s: %s", e.StatusCode, e.Code, e.Reason)
}

// RejectionError is a well-formed SEFAZ answer whose cStat means the request
// was refused, such as 593 (CNPJ-Base differs from the certificate).
type RejectionError struct {
	CStat   int
	XMotivo string
}

func (e *RejectionError) Error() string {
	return fmt.Sprintf("SEFAZ rejected the request: cStat %d: %s", e.CStat, e.XMotivo)
}

// ClientConfig configures a Client. Environment and Certificate are required.
type ClientConfig struct {
	Environment nfse.Environment
	Certificate *tls.Certificate
	// RootCAs verifies the SEFAZ servers; nil uses the platform roots, which
	// already trust the public chains the Ambiente Nacional uses.
	RootCAs *x509.CertPool
	// Timeout bounds each HTTP attempt; zero means DefaultTimeout.
	Timeout time.Duration
	Log     *slog.Logger
	// HTTPClient is copied and gets the mTLS transport; nil uses a new client.
	HTTPClient *http.Client
	// Endpoints overrides the URLs chosen from Environment.
	Endpoints *Endpoints
}

// Client calls the NF-e web services of the Ambiente Nacional.
type Client struct {
	http      *httpclient.Client
	tpAmb     string
	endpoints Endpoints
	timeout   time.Duration
	// tlsConfig is the TLS configuration of the HTTP transport, kept for
	// CheckTLS.
	tlsConfig *tls.Config
	log       *slog.Logger
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Certificate == nil {
		return nil, errors.New("certificate is required")
	}
	tpAmb, err := TpAmb(cfg.Environment)
	if err != nil {
		return nil, err
	}
	endpoints, err := EndpointsFor(cfg.Environment)
	if err != nil {
		return nil, err
	}
	if cfg.Endpoints != nil {
		endpoints = *cfg.Endpoints
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	httpClient, err := httpclient.New(httpclient.Config{
		Certificate: cfg.Certificate,
		RootCAs:     cfg.RootCAs,
		HTTPClient:  cfg.HTTPClient,
		Timeout:     timeout,
		// Every distribution request counts against the SEFAZ hourly limit,
		// so the transport never retries on its own; post decides per call.
		Retry:      httpclient.RetryConfig{MaxRetries: 0},
		Log:        cfg.Log,
		LogLabel:   "SEFAZ",
		RedactBody: redact.MaskXMLIdentifiers,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		http:      httpClient,
		tpAmb:     tpAmb,
		endpoints: endpoints,
		timeout:   timeout,
		tlsConfig: httpclient.NewTransport(cfg.Certificate, cfg.RootCAs).TLSClientConfig,
		log:       cfg.Log,
	}, nil
}

// envelope wraps body, which is embedded verbatim, in a SOAP 1.2 envelope
// without a Header.
func envelope(body []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<soap:Envelope xmlns:soap="` + soapNamespace + `"><soap:Body>`)
	b.Write(body)
	b.WriteString(`</soap:Body></soap:Envelope>`)
	return b.Bytes()
}

// contentType carries the SOAP 1.2 action; SOAP 1.2 has no SOAPAction header.
func contentType(action string) string {
	return `application/soap+xml; charset=utf-8; action="` + action + `"`
}

// post sends body in a SOAP envelope and returns the response body.
// transportRetries is how many times a request that got no HTTP response at
// all is sent again; HTTP errors are never retried.
func (c *Client) post(ctx context.Context, url, action string, body []byte, transportRetries int) ([]byte, error) {
	req := httpclient.Request{
		Method:   http.MethodPost,
		URL:      url,
		Header:   http.Header{"Content-Type": {contentType(action)}},
		Body:     envelope(body),
		MaxBytes: MaxResponseBytes,
		// SOAP faults come with HTTP 500 and carry the reason in the body.
		Expect: func(status int) bool { return status == http.StatusInternalServerError },
	}

	resp, err := c.http.Do(ctx, req)
	for retry := 0; retry < transportRetries && isTransportError(err); retry++ {
		resp, err = c.http.Do(ctx, req)
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusInternalServerError {
		// httpclient logged the accepted 500 at Debug only.
		c.logErrorResponse(ctx, url, resp.StatusCode, resp.Body)
		if fault, ok := parseFault(resp.Body); ok {
			fault.StatusCode = resp.StatusCode
			return nil, &fault
		}
		return nil, c.http.NewStatusError(req.Method, url, resp.StatusCode, resp.Body)
	}
	return resp.Body, nil
}

// logErrorResponse logs an HTTP 500 answer at Error level with its body
// redacted, as httpclient does for statuses it does not accept.
func (c *Client) logErrorResponse(ctx context.Context, url string, status int, body []byte) {
	if c.log == nil {
		return
	}
	c.log.ErrorContext(ctx, "SEFAZ Error Response",
		slog.String("method", http.MethodPost),
		slog.String("url", url),
		slog.Int("status", status),
		slog.String("body", httpclient.TruncateForLog(redact.MaskXMLIdentifiers(body), httpclient.MaxErrorLogBodyBytes)))
}

// isTransportError reports whether err means no HTTP response arrived.
func isTransportError(err error) bool {
	var statusErr *httpclient.StatusError
	return errors.As(err, &statusErr) && statusErr.StatusCode == 0
}

// soapFault covers SOAP 1.2 (Code/Value, Reason/Text) and SOAP 1.1
// (faultcode, faultstring). Tags without a namespace match any prefix.
type soapFault struct {
	Code        string `xml:"Code>Value"`
	Reason      string `xml:"Reason>Text"`
	FaultCode   string `xml:"faultcode"`
	FaultString string `xml:"faultstring"`
}

// parseFault reads the Fault in body; ok is false when there is none.
func parseFault(body []byte) (fault FaultError, ok bool) {
	var f soapFault
	if err := decodeElement(body, "Fault", &f); err != nil {
		return FaultError{}, false
	}
	fault = FaultError{Code: f.Code, Reason: f.Reason}
	if fault.Code == "" {
		fault.Code = f.FaultCode
	}
	if fault.Reason == "" {
		fault.Reason = f.FaultString
	}
	fault.Code = strings.TrimSpace(fault.Code)
	fault.Reason = strings.TrimSpace(fault.Reason)
	return fault, true
}

// findElements returns the raw bytes of every element named local, whatever
// its namespace prefix, in document order. Elements nested inside a match
// are not reported separately.
func findElements(body []byte, local string) ([][]byte, error) {
	d := xml.NewDecoder(bytes.NewReader(body))
	var found [][]byte
	for {
		start := d.InputOffset()
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return found, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parse SEFAZ response: %w", err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Local != local {
			continue
		}
		if err := d.Skip(); err != nil {
			return nil, fmt.Errorf("parse SEFAZ response: %w", err)
		}
		found = append(found, body[start:d.InputOffset()])
	}
}

// decodeElement decodes the first element named local, whatever its
// namespace prefix, into v.
func decodeElement(body []byte, local string, v any) error {
	d := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("SEFAZ response has no %s element", local)
		}
		if err != nil {
			return fmt.Errorf("parse SEFAZ response: %w", err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Local != local {
			continue
		}
		if err := d.DecodeElement(v, &el); err != nil {
			return fmt.Errorf("parse %s: %w", local, err)
		}
		return nil
	}
}
