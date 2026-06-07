package adn

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"log/slog"

	"github.com/sethvargo/go-retry"

	"github.com/vasfvitor/nanci/internal/nfse"
)

const (
	BaseURLProduction           = "https://adn.nfse.gov.br"
	BaseURLRestrictedProduction = "https://adn.producaorestrita.nfse.gov.br"

	MaxJSONResponseBytes = 20 * 1024 * 1024 // 20 MiB
	MaxErrorBodyBytes    = 64 * 1024        // 64 KiB
)

type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
	Retryable  bool
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
	Environment nfse.Environment
	HTTPClient  *http.Client
	Certificate *tls.Certificate
	Retry       RetryConfig
	Log         *slog.Logger
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	backoff    retry.Backoff
	log        *slog.Logger
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Retry.MaxRetries < 0 {
		return nil, fmt.Errorf("max retries must not be negative")
	}

	var baseURLStr string
	switch cfg.Environment {
	case nfse.EnvironmentProduction:
		baseURLStr = BaseURLProduction
	case nfse.EnvironmentRestricted:
		baseURLStr = BaseURLRestrictedProduction
	default:
		return nil, fmt.Errorf("invalid environment: %s", cfg.Environment)
	}

	u, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Clone the default transport to preserve proxy/dial settings
	transport := http.DefaultTransport.(*http.Transport).Clone()

	var tlsConfig *tls.Config
	if cfg.Certificate != nil {
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{*cfg.Certificate},
			MinVersion:   tls.VersionTLS12,
		}
	} else {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	transport.TLSClientConfig = tlsConfig

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	client.Transport = transport

	b := retry.NewExponential(cfg.Retry.Initial)
	b = retry.WithMaxRetries(uint64(cfg.Retry.MaxRetries), b)
	b = retry.WithCappedDuration(cfg.Retry.MaxDelay, b)
	b = retry.WithJitterPercent(20, b)

	return &Client{
		baseURL:    u,
		httpClient: client,
		backoff:    b,
		log:        cfg.Log,
	}, nil
}

func (c *Client) request(ctx context.Context, method, path string, bodyProvider func() io.Reader, dest interface{}) error {
	u := c.baseURL.ResolveReference(&url.URL{Path: path}).String()

	return retry.Do(ctx, c.backoff, func(ctx context.Context) error {
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
			return fmt.Errorf("failed to create request: %w", err) // Not retryable
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Transport error - retryable if context is active
			if ctx.Err() != nil {
				return err // Context canceled
			}
			return retry.RetryableError(fmt.Errorf("transport error: %w", err))
		}
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
					// JSON decode error - not retryable
					return fmt.Errorf("failed to decode json response: %w", err)
				}
			}
			return nil
		}

		// Read error response body bounded
		errBodyReader := io.LimitReader(resp.Body, MaxErrorBodyBytes)
		errBodyBytes, _ := io.ReadAll(errBodyReader)
		
		if c.log != nil {
			c.log.ErrorContext(ctx, "ADN API Error Response", slog.String("method", method), slog.String("path", path), slog.Int("status", resp.StatusCode), slog.String("body", string(errBodyBytes)), slog.Duration("latency", time.Since(start)))
		}

		apiErr := &APIError{
			Method:     method,
			URL:        u,
			StatusCode: resp.StatusCode,
			Body:       string(errBodyBytes),
		}

		// Handle Retry-After logic if we want, go-retry doesn't inherently parse the header.
		// We'll let go-retry handle backoff, but determine retryability
		if isRetryableStatus(resp.StatusCode) {
			apiErr.Retryable = true
			return retry.RetryableError(apiErr)
		}

		apiErr.Retryable = false
		return apiErr // Not retryable
	})
}

func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || // 408
		status == http.StatusTooEarly || // 425
		status == http.StatusTooManyRequests || // 429
		(status >= 500 && status <= 599) // 5xx
}
