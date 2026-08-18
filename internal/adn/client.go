package adn

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sethvargo/go-retry"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

const (
	BaseURLProduction           = "https://adn.nfse.gov.br/contribuintes"
	BaseURLRestrictedProduction = "https://adn.producaorestrita.nfse.gov.br/contribuintes"

	MaxJSONResponseBytes = 20 * 1024 * 1024 // 20 MiB
	MaxErrorBodyBytes    = 64 * 1024        // 64 KiB

	// maxErrorLogBodyBytes caps the error body attached to Error-level log
	// records; the full body (up to MaxErrorBodyBytes) is only logged at trace.
	maxErrorLogBodyBytes = 2 * 1024
	// maxTraceBodyBytes caps response bodies logged at trace level.
	maxTraceBodyBytes = 256 * 1024
)

type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
	Retryable  bool
	RetryAfter time.Duration
}

type responseError struct {
	Codigo string `json:"Codigo"`
}

type noDocumentsResponse struct {
	StatusProcessamento string            `json:"StatusProcessamento"`
	LoteDFe             []json.RawMessage `json:"LoteDFe"`
	Erros               []responseError   `json:"Erros"`
}

// Error keeps the message bounded: the full body stays in Body, but the
// message (which ends up in logs and the UI) only carries a prefix.
func (e *APIError) Error() string {
	return fmt.Sprintf("ADN API error %s %s: status %d, body: %s", e.Method, e.URL, e.StatusCode, truncateForLog([]byte(e.Body), maxErrorLogBodyBytes))
}

// RawGet performs a GET request to an arbitrary relative path and decodes the JSON into dest.
func (c *Client) RawGet(ctx context.Context, path string, dest any) error {
	return c.request(ctx, "GET", path, nil, dest)
}

type RetryConfig struct {
	MaxRetries int
	Initial    time.Duration
	MaxDelay   time.Duration
}

type ClientConfig struct {
	BaseURL     string
	HTTPClient  *http.Client
	Certificate *tls.Certificate
	Retry       RetryConfig
	Log         *slog.Logger
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	retry      RetryConfig
	log        *slog.Logger
}

var ErrNoDocumentsLocated = errors.New("nenhum documento localizado")

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Retry.MaxRetries < 0 {
		return nil, fmt.Errorf("max retries must not be negative")
	}

	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	if u.Path == "" {
		u.Path = "/"
	} else if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	// Clone the default transport to preserve proxy/dial settings
	transport := http.DefaultTransport.(*http.Transport).Clone()

	var tlsConfig *tls.Config
	if cfg.Certificate != nil {
		tlsConfig = &tls.Config{
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return cfg.Certificate, nil
			},
			MinVersion:    tls.VersionTLS12,
			Renegotiation: tls.RenegotiateFreelyAsClient,
		}
	} else {
		tlsConfig = &tls.Config{
			MinVersion:    tls.VersionTLS12,
			Renegotiation: tls.RenegotiateFreelyAsClient,
		}
	}
	transport.TLSClientConfig = tlsConfig
	transport.ForceAttemptHTTP2 = false

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	client.Transport = transport

	initial := cfg.Retry.Initial
	if initial <= 0 {
		initial = 1 * time.Second
	}
	maxDelay := cfg.Retry.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	maxRetries := cfg.Retry.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	return &Client{
		baseURL:    u,
		httpClient: client,
		retry: RetryConfig{
			MaxRetries: maxRetries,
			Initial:    initial,
			MaxDelay:   maxDelay,
		},
		log: cfg.Log,
	}, nil
}

func (c *Client) request(ctx context.Context, method, path string, bodyProvider func() io.Reader, dest any) error {
	rel, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err) // Not retryable
	}
	u := c.baseURL.ResolveReference(rel).String()

	backoff := c.newBackoff()
	for {
		start := time.Now()
		if c.log != nil {
			c.log.Log(ctx, logger.LevelTrace, "ADN API Request", slog.String("method", method), slog.String("path", sanitizeURL(path)))
		}

		var reqBody io.Reader
		if bodyProvider != nil {
			reqBody = bodyProvider()
		}

		req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return err
			}
			if retryErr := c.waitForRetry(ctx, backoff, &APIError{
				Method:    method,
				URL:       sanitizeURL(u),
				Body:      fmt.Sprintf("transport error: %v", err),
				Retryable: true,
			}); retryErr != nil {
				return retryErr
			}
			continue
		}

		err = func() error {
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, MaxJSONResponseBytes+1))

				if c.log != nil {
					// Bodies carry base64 XML with fiscal data; keep them out of
					// debug logs and only expose them at trace level.
					c.log.DebugContext(ctx, "ADN API Response", slog.String("method", method), slog.String("url", sanitizeURL(u)), slog.Int("status", resp.StatusCode), slog.Int("body_bytes", len(bodyBytes)), slog.Duration("latency", time.Since(start)))
					if c.log.Enabled(ctx, logger.LevelTrace) {
						c.log.Log(ctx, logger.LevelTrace, "ADN API Response Body", slog.String("method", method), slog.String("url", sanitizeURL(u)), slog.String("body", truncateForLog(bodyBytes, maxTraceBodyBytes)))
					}
				}
				if dest != nil {
					if err := json.Unmarshal(bodyBytes, dest); err != nil {
						return fmt.Errorf("failed to decode json response: %w", err)
					}
				}
				return nil
			}

			errBodyReader := io.LimitReader(resp.Body, MaxErrorBodyBytes)
			errBodyBytes, _ := io.ReadAll(errBodyReader)

			if resp.StatusCode == http.StatusNotFound {
				if isNoDocumentsResponse(errBodyBytes) {
					if c.log != nil {
						c.log.DebugContext(ctx, "ADN API empty result", slog.String("method", method), slog.String("path", sanitizeURL(path)), slog.Int("status", resp.StatusCode))
					}
					return ErrNoDocumentsLocated
				}
				if explicit404 := classifyUnexpected404Body(errBodyBytes); explicit404 != "" {
					return c.newAPIError(method, u, resp.StatusCode, explicit404, false, 0)
				}
			}

			if c.log != nil {
				c.log.ErrorContext(ctx, "ADN API Error Response", slog.String("method", method), slog.String("path", sanitizeURL(path)), slog.Int("status", resp.StatusCode), slog.String("body", truncateForLog(errBodyBytes, maxErrorLogBodyBytes)), slog.Duration("latency", time.Since(start)))
				if len(errBodyBytes) > maxErrorLogBodyBytes && c.log.Enabled(ctx, logger.LevelTrace) {
					c.log.Log(ctx, logger.LevelTrace, "ADN API Error Response Body", slog.String("method", method), slog.String("path", sanitizeURL(path)), slog.String("body", truncateForLog(errBodyBytes, maxTraceBodyBytes)))
				}
			}

			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.retry.MaxDelay)
			return c.newAPIError(method, u, resp.StatusCode, string(errBodyBytes), isRetryableStatus(resp.StatusCode), retryAfter)
		}()
		if err == nil {
			return nil
		}

		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Retryable {
			if retryErr := c.waitForRetry(ctx, backoff, apiErr); retryErr != nil {
				return retryErr
			}
			continue
		}
		return err
	}
}

func (c *Client) newBackoff() retry.Backoff {
	b := retry.NewExponential(c.retry.Initial)
	b = retry.WithMaxRetries(uint64(c.retry.MaxRetries), b) //nolint:gosec // intentional: max retries is known to be non-negative
	b = retry.WithCappedDuration(c.retry.MaxDelay, b)
	b = retry.WithJitterPercent(20, b)
	return b
}

func (c *Client) waitForRetry(ctx context.Context, backoff retry.Backoff, apiErr *APIError) error {
	delay, stop := backoff.Next()
	if stop {
		return apiErr
	}
	if apiErr.RetryAfter > 0 {
		delay = apiErr.RetryAfter
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) newAPIError(method, url string, statusCode int, body string, retryable bool, retryAfter time.Duration) *APIError {
	return &APIError{
		Method:     method,
		URL:        sanitizeURL(url),
		StatusCode: statusCode,
		Body:       body,
		Retryable:  retryable,
		RetryAfter: retryAfter,
	}
}

func isNoDocumentsResponse(body []byte) bool {
	var response noDocumentsResponse
	_ = json.Unmarshal(body, &response)
	if response.StatusProcessamento == "" {
		return false
	}
	if response.StatusProcessamento != "NENHUM_DOCUMENTO_LOCALIZADO" {
		return false
	}
	if len(response.Erros) != 1 || response.Erros[0].Codigo != "E2220" {
		return false
	}
	return true
}

func classifyUnexpected404Body(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "unexpected 404 from ADN route: empty response body"
	}
	if strings.HasPrefix(trimmed, "<") {
		return fmt.Sprintf("unexpected 404 from ADN route: non-ADN HTML response: %s", trimmed)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Sprintf("unexpected 404 from ADN route: non-ADN payload: %s", trimmed)
	}
	if _, ok := payload["StatusProcessamento"]; ok {
		return ""
	}
	if _, ok := payload["LoteDFe"]; ok {
		return ""
	}
	if _, ok := payload["Erros"]; ok {
		return ""
	}
	return fmt.Sprintf("unexpected 404 from ADN route: payload does not match ADN envelope: %s", trimmed)
}

func parseRetryAfter(raw string, maxDelay time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0
	}
	delay := time.Duration(seconds) * time.Second
	if maxDelay > 0 && delay > maxDelay {
		return maxDelay
	}
	return delay
}

func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || // 408
		status == http.StatusTooEarly || // 425
		status == http.StatusTooManyRequests || // 429
		(status >= 500 && status <= 599) // 5xx
}

// truncateForLog shortens body for log output without splitting a UTF-8
// sequence, marking the cut so readers know the record is partial.
func truncateForLog(body []byte, limit int) string {
	if len(body) <= limit {
		return string(body)
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return string(body[:cut]) + "... (truncated)"
}

// sanitizeURL masks the cnpjConsulta query parameter so log records do not
// carry the consulted CNPJ in clear. Other parts of the URL are unchanged.
func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	v := q.Get("cnpjConsulta")
	if v == "" {
		return raw
	}
	q.Set("cnpjConsulta", maskIdentifier(v))
	// Encode percent-escapes '*'; keep the mask readable in log output.
	u.RawQuery = strings.ReplaceAll(q.Encode(), "%2A", "*")
	return u.String()
}

// maskIdentifier keeps the first and last two characters of v and stars the
// rest, so masked values stay recognisable without being reusable.
func maskIdentifier(v string) string {
	if len(v) <= 4 {
		return strings.Repeat("*", len(v))
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}
