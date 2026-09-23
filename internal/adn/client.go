package adn

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
)

const (
	BaseURLProduction           = "https://adn.nfse.gov.br/contribuintes"
	BaseURLRestrictedProduction = "https://adn.producaorestrita.nfse.gov.br/contribuintes"

	MaxJSONResponseBytes = 20 * 1024 * 1024 // 20 MiB

	// maxErrorLogBodyBytes caps the error body attached to Error-level log
	// records; the full body is only logged at trace.
	maxErrorLogBodyBytes = httpclient.MaxErrorLogBodyBytes
)

// APIError is the error returned for a rejected ADN response.
type APIError = httpclient.StatusError

type responseError struct {
	Codigo string `json:"Codigo"`
}

type noDocumentsResponse struct {
	StatusProcessamento string            `json:"StatusProcessamento"`
	LoteDFe             []json.RawMessage `json:"LoteDFe"`
	Erros               []responseError   `json:"Erros"`
}

// RawGet performs a GET request to an arbitrary relative path and decodes the JSON into dest.
func (c *Client) RawGet(ctx context.Context, path string, dest any) error {
	return c.request(ctx, "GET", path, dest)
}

type RetryConfig struct {
	// MaxRetries is the number of retries after the first attempt. Zero
	// means the ADN default of 3.
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
	httpClient *httpclient.Client
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

	maxRetries := cfg.Retry.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	httpClient, err := httpclient.New(httpclient.Config{
		Certificate: cfg.Certificate,
		HTTPClient:  cfg.HTTPClient,
		Retry: httpclient.RetryConfig{
			MaxRetries: maxRetries,
			Initial:    cfg.Retry.Initial,
			MaxDelay:   cfg.Retry.MaxDelay,
		},
		Log:       cfg.Log,
		LogLabel:  "ADN API",
		RedactURL: sanitizeURL,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:    u,
		httpClient: httpClient,
		log:        cfg.Log,
	}, nil
}

func (c *Client) request(ctx context.Context, method, path string, dest any) error {
	rel, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err) // Not retryable
	}
	u := c.baseURL.ResolveReference(rel).String()

	resp, err := c.httpClient.Do(ctx, httpclient.Request{
		Method: method,
		URL:    u,
		Header: http.Header{
			"Content-Type": {"application/json"},
			"Accept":       {"application/json"},
		},
		MaxBytes: MaxJSONResponseBytes,
		// ADN answers an empty queue with 404; notFound tells it apart
		// from a real routing error.
		Expect: func(status int) bool { return status == http.StatusNotFound },
	})
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return c.notFound(ctx, method, path, u, resp.Body)
	}

	if dest != nil {
		if err := json.Unmarshal(resp.Body, dest); err != nil {
			return fmt.Errorf("failed to decode json response: %w", err)
		}
	}
	return nil
}

// notFound classifies a 404: the ADN "no documents" envelope is
// ErrNoDocumentsLocated, anything else is an APIError.
func (c *Client) notFound(ctx context.Context, method, path, u string, body []byte) error {
	if isNoDocumentsResponse(body) {
		if c.log != nil {
			c.log.DebugContext(ctx, "ADN API empty result", slog.String("method", method), slog.String("path", sanitizeURL(path)), slog.Int("status", http.StatusNotFound))
		}
		return ErrNoDocumentsLocated
	}
	if explicit404 := classifyUnexpected404Body(body); explicit404 != "" {
		return c.httpClient.NewStatusError(method, u, http.StatusNotFound, []byte(explicit404))
	}

	if c.log != nil {
		c.log.ErrorContext(ctx, "ADN API Error Response", slog.String("method", method), slog.String("path", sanitizeURL(path)), slog.Int("status", http.StatusNotFound), slog.String("body", httpclient.TruncateForLog(body, maxErrorLogBodyBytes)))
	}
	return c.httpClient.NewStatusError(method, u, http.StatusNotFound, body)
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
	q.Set("cnpjConsulta", httpclient.MaskIdentifier(v))
	// Encode percent-escapes '*'; keep the mask readable in log output.
	u.RawQuery = strings.ReplaceAll(q.Encode(), "%2A", "*")
	return u.String()
}
