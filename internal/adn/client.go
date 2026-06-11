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

	"github.com/sethvargo/go-retry"

	"github.com/vasfvitor/nanci/internal/nfse"
)

const (
	BaseURLProduction           = "https://adn.nfse.gov.br/contribuintes"
	BaseURLRestrictedProduction = "https://adn.producaorestrita.nfse.gov.br/contribuintes"

	MaxJSONResponseBytes = 20 * 1024 * 1024 // 20 MiB
	MaxErrorBodyBytes    = 64 * 1024        // 64 KiB
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

func (e *APIError) Error() string {
	return fmt.Errorf("ADN API error %s %s: status %d, body: %s", e.Method, e.URL, e.StatusCode, e.Body).Error()
}

type RetryConfig struct {
	MaxRetries int
	Initial    time.Duration
	MaxDelay   time.Duration
}

type ClientConfig struct {
	Environment     nfse.Environment
	BaseURLOverride string
	HTTPClient      *http.Client
	Certificate     *tls.Certificate
	Retry           RetryConfig
	Log             *slog.Logger
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	retry      RetryConfig
	log        *slog.Logger
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Retry.MaxRetries < 0 {
		return nil, fmt.Errorf("max retries must not be negative")
	}

	var baseURLStr string
	if cfg.BaseURLOverride != "" {
		baseURLStr = cfg.BaseURLOverride
	} else {
		switch cfg.Environment {
		case nfse.EnvironmentProduction:
			baseURLStr = BaseURLProduction
		case nfse.EnvironmentRestricted:
			baseURLStr = BaseURLRestrictedProduction
		default:
			return nil, fmt.Errorf("invalid environment: %s", cfg.Environment)
		}
	}

	u, err := url.Parse(baseURLStr)
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

func (c *Client) request(ctx context.Context, method, path string, bodyProvider func() io.Reader, dest interface{}) error {
	rel, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err) // Not retryable
	}
	u := c.baseURL.ResolveReference(rel).String()

	backoff := c.newBackoff()
	for {
		start := time.Now()
		if c.log != nil {
			c.log.Log(ctx, slog.Level(-8), "ADN API Request", slog.String("method", method), slog.String("path", path))
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
				URL:       u,
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
				if c.log != nil {
					c.log.DebugContext(ctx, "ADN API Response", slog.String("method", method), slog.String("path", path), slog.Int("status", resp.StatusCode), slog.Duration("latency", time.Since(start)))
				}
				if dest != nil {
					lr := io.LimitReader(resp.Body, MaxJSONResponseBytes+1)
					if err := json.NewDecoder(lr).Decode(dest); err != nil {
						return fmt.Errorf("failed to decode json response: %w", err)
					}
				}
				return nil
			}

			errBodyReader := io.LimitReader(resp.Body, MaxErrorBodyBytes)
			errBodyBytes, _ := io.ReadAll(errBodyReader)

			if resp.StatusCode == http.StatusNotFound && dest != nil {
				handled, err := tryDecodeNoDocumentsResponse(errBodyBytes, dest)
				if err != nil {
					return c.newAPIError(method, u, resp.StatusCode, err.Error(), false, 0)
				}
				if handled {
					if c.log != nil {
						c.log.DebugContext(ctx, "ADN API empty result", slog.String("method", method), slog.String("path", path), slog.Int("status", resp.StatusCode))
					}
					return nil
				}
				if explicit404 := classifyUnexpected404Body(errBodyBytes); explicit404 != "" {
					return c.newAPIError(method, u, resp.StatusCode, explicit404, false, 0)
				}
			}

			if c.log != nil {
				c.log.ErrorContext(ctx, "ADN API Error Response", slog.String("method", method), slog.String("path", path), slog.Int("status", resp.StatusCode), slog.String("body", string(errBodyBytes)), slog.Duration("latency", time.Since(start)))
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
	b = retry.WithMaxRetries(uint64(c.retry.MaxRetries), b)
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
		URL:        url,
		StatusCode: statusCode,
		Body:       body,
		Retryable:  retryable,
		RetryAfter: retryAfter,
	}
}

func tryDecodeNoDocumentsResponse(body []byte, dest interface{}) (bool, error) {
	var response noDocumentsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return false, nil
	}
	if response.StatusProcessamento != "NENHUM_DOCUMENTO_LOCALIZADO" {
		return false, nil
	}
	if len(response.Erros) != 1 || response.Erros[0].Codigo != "E2220" {
		return false, nil
	}

	documentResponse, ok := dest.(*DocumentResponse)
	if !ok {
		return false, nil
	}

	documentResponse.UltNSU = 0
	documentResponse.MaxNSU = 0
	documentResponse.Docs = nil
	return true, nil
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
