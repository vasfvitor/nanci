package adn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/vasfvitor/nanci/internal/foundation/retry"
)

const (
	BaseURLProduction           = "https://adn.nfse.gov.br"
	BaseURLRestrictedProduction = "https://adn.producaorestrita.nfse.gov.br"

	EnvProduction           = "producao"
	EnvRestrictedProduction = "producao_restrita"
)

// Client is the main API client for the ADN web services.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new ADN API client for the specified environment.
func NewClient(httpClient *http.Client, env string) (*Client, error) {
	var baseURL string
	switch env {
	case EnvProduction:
		baseURL = BaseURLProduction
	case EnvRestrictedProduction:
		baseURL = BaseURLRestrictedProduction
	default:
		return nil, fmt.Errorf("invalid environment: %s", env)
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// Request performs an HTTP request with exponential backoff.
// It parses the JSON response into the 'dest' interface if provided.
func (c *Client) Request(ctx context.Context, method, path string, body interface{}, dest interface{}) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("failed to parse url: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute with retry
	return retry.Do(ctx, retry.DefaultConfig(), func() error {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("http request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			if dest != nil {
				if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
					return fmt.Errorf("failed to decode response: %w", err)
				}
			}
			return nil
		}

		// Read error response body
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(errorBody))
	})
}
