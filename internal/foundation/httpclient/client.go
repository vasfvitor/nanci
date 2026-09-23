// Package httpclient is the mTLS HTTP client shared by the tax authority web
// service clients. It owns the transport, the retry loop, bounded body reads
// and the request/response log records; callers own headers, payload
// encoding and the meaning of each status.
package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

// DefaultMaxBytes caps a response body when Request.MaxBytes is not set.
const DefaultMaxBytes = 20 * 1024 * 1024 // 20 MiB

// ErrResponseTooLarge is returned when a response body exceeds Request.MaxBytes.
var ErrResponseTooLarge = errors.New("response body exceeds size limit")

type Config struct {
	Certificate *tls.Certificate
	// RootCAs is the pool used to verify the server; nil uses the platform roots.
	RootCAs *x509.CertPool
	// HTTPClient is copied and its Transport replaced; nil uses a new client.
	HTTPClient *http.Client
	// Timeout bounds each attempt, including reading the body; zero means none.
	Timeout time.Duration
	Retry   RetryConfig
	Log     *slog.Logger
	// LogLabel prefixes log messages and errors, for example "ADN API".
	LogLabel string
	// RedactURL masks identifiers in URLs before they reach a log record or an error.
	RedactURL func(string) string
	// RedactBody masks identifiers in bodies before they reach a log record or
	// StatusError.Error(). StatusError.Body stays raw.
	RedactBody func([]byte) []byte
}

type Request struct {
	Method string
	// URL is absolute.
	URL    string
	Header http.Header
	// Body is sent again on every attempt.
	Body []byte
	// MaxBytes caps the accepted response body; a larger body is an error.
	// Zero means DefaultMaxBytes.
	MaxBytes int64
	// Expect lists non-2xx statuses the caller handles itself: they are
	// returned as a Response instead of a StatusError, are not retried, and
	// are not logged as errors.
	Expect func(status int) bool
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// StatusError describes a rejected response or a transport failure
// (StatusCode 0).
type StatusError struct {
	Method string
	// URL is already redacted.
	URL        string
	StatusCode int
	// Body is the raw body, bounded by MaxErrorBodyBytes, for classification.
	Body       string
	Retryable  bool
	RetryAfter time.Duration

	label   string
	logBody string
}

// Error keeps the message bounded and redacted: the full body stays in Body,
// but the message (which ends up in logs and the UI) only carries a prefix.
func (e *StatusError) Error() string {
	return fmt.Sprintf("%s error %s %s: status %d, body: %s", e.label, e.Method, e.URL, e.StatusCode, e.logBody)
}

type Client struct {
	httpClient *http.Client
	retry      RetryConfig
	log        *slog.Logger
	label      string
	redactURL  func(string) string
	redactBody func([]byte) []byte
}

func New(cfg Config) (*Client, error) {
	if cfg.Retry.MaxRetries < 0 {
		return nil, fmt.Errorf("max retries must not be negative")
	}

	httpClient := &http.Client{}
	if cfg.HTTPClient != nil {
		copied := *cfg.HTTPClient
		httpClient = &copied
	}
	httpClient.Transport = NewTransport(cfg.Certificate, cfg.RootCAs)
	if cfg.Timeout > 0 {
		httpClient.Timeout = cfg.Timeout
	}

	retryCfg := cfg.Retry
	if retryCfg.Initial <= 0 {
		retryCfg.Initial = 1 * time.Second
	}
	if retryCfg.MaxDelay <= 0 {
		retryCfg.MaxDelay = 30 * time.Second
	}

	label := cfg.LogLabel
	if label == "" {
		label = "HTTP"
	}
	redactURL := cfg.RedactURL
	if redactURL == nil {
		redactURL = func(s string) string { return s }
	}
	redactBody := cfg.RedactBody
	if redactBody == nil {
		redactBody = func(b []byte) []byte { return b }
	}

	return &Client{
		httpClient: httpClient,
		retry:      retryCfg,
		log:        cfg.Log,
		label:      label,
		redactURL:  redactURL,
		redactBody: redactBody,
	}, nil
}

// Do sends req, retrying transport failures and retryable statuses
// (408, 425, 429, 5xx) with exponential backoff or the server's Retry-After.
// A 2xx or expected status comes back as a Response; any other status comes
// back as a *StatusError.
func (c *Client) Do(ctx context.Context, req Request) (Response, error) {
	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	logURL := c.redactURL(req.URL)

	backoff := newBackoff(c.retry)
	for {
		resp, err := c.attempt(ctx, req, logURL, maxBytes)
		if err == nil {
			return resp, nil
		}

		var statusErr *StatusError
		if errors.As(err, &statusErr) && statusErr.Retryable {
			if retryErr := waitForRetry(ctx, backoff, statusErr); retryErr != nil {
				return Response{}, retryErr
			}
			continue
		}
		return Response{}, err
	}
}

// NewStatusError builds the error Do would return for statusCode and body.
// Callers use it when a status accepted through Request.Expect turns out to
// be a failure after all.
func (c *Client) NewStatusError(method, rawURL string, statusCode int, body []byte) *StatusError {
	return c.statusError(method, c.redactURL(rawURL), statusCode, body)
}

func (c *Client) attempt(ctx context.Context, req Request, logURL string, maxBytes int64) (Response, error) {
	start := time.Now()
	if c.log != nil {
		c.log.Log(ctx, logger.LevelTrace, c.label+" Request", slog.String("method", req.Method), slog.String("url", logURL))
	}

	var body io.Reader
	if req.Body != nil {
		body = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return Response{}, fmt.Errorf("failed to create request: %w", err)
	}
	if req.Header != nil {
		httpReq.Header = req.Header.Clone()
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, err
		}
		statusErr := c.statusError(req.Method, logURL, 0, []byte(fmt.Sprintf("transport error: %v", err)))
		statusErr.Retryable = true
		return Response{}, statusErr
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	accepted := resp.StatusCode >= 200 && resp.StatusCode <= 299
	if !accepted && req.Expect != nil {
		accepted = req.Expect(resp.StatusCode)
	}
	if accepted {
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
		if err != nil {
			return Response{}, fmt.Errorf("%s %s %s: failed to read response body: %w", c.label, req.Method, logURL, err)
		}
		if int64(len(respBody)) > maxBytes {
			return Response{}, fmt.Errorf("%s %s %s: %w (limit %d bytes)", c.label, req.Method, logURL, ErrResponseTooLarge, maxBytes)
		}

		if c.log != nil {
			// Bodies carry fiscal data; keep them out of debug logs and only
			// expose them at trace level.
			c.log.DebugContext(ctx, c.label+" Response", slog.String("method", req.Method), slog.String("url", logURL), slog.Int("status", resp.StatusCode), slog.Int("body_bytes", len(respBody)), slog.Duration("latency", time.Since(start)))
			if c.log.Enabled(ctx, logger.LevelTrace) {
				c.log.Log(ctx, logger.LevelTrace, c.label+" Response Body", slog.String("method", req.Method), slog.String("url", logURL), slog.String("body", TruncateForLog(c.redactBody(respBody), maxTraceBodyBytes)))
			}
		}
		return Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: respBody}, nil
	}

	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, MaxErrorBodyBytes))

	if c.log != nil {
		logBody := c.redactBody(errBody)
		c.log.ErrorContext(ctx, c.label+" Error Response", slog.String("method", req.Method), slog.String("url", logURL), slog.Int("status", resp.StatusCode), slog.String("body", TruncateForLog(logBody, MaxErrorLogBodyBytes)), slog.Duration("latency", time.Since(start)))
		if len(logBody) > MaxErrorLogBodyBytes && c.log.Enabled(ctx, logger.LevelTrace) {
			c.log.Log(ctx, logger.LevelTrace, c.label+" Error Response Body", slog.String("method", req.Method), slog.String("url", logURL), slog.String("body", TruncateForLog(logBody, maxTraceBodyBytes)))
		}
	}

	statusErr := c.statusError(req.Method, logURL, resp.StatusCode, errBody)
	statusErr.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), c.retry.MaxDelay)
	return Response{}, statusErr
}

func (c *Client) statusError(method, logURL string, statusCode int, body []byte) *StatusError {
	return &StatusError{
		Method:     method,
		URL:        logURL,
		StatusCode: statusCode,
		Body:       string(body),
		Retryable:  isRetryableStatus(statusCode),
		label:      c.label,
		logBody:    TruncateForLog(c.redactBody(body), MaxErrorLogBodyBytes),
	}
}
