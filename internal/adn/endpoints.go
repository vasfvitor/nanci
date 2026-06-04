package adn

import (
	"context"
	"fmt"
)

// DocumentResponse represents the API response for the GET /DFe endpoint.
type DocumentResponse struct {
	UltNSU int64              `json:"ultNSU"`
	MaxNSU int64              `json:"maxNSU"`
	Docs   []DocumentEnvelope `json:"docFisc"`
}

// DocumentEnvelope wraps a single fiscal document.
type DocumentEnvelope struct {
	NSU           int64  `json:"nsu"`
	Schema        string `json:"schema"`         // e.g., "procNFSe_v1.00.xsd"
	XMLGZipBase64 string `json:"nfseXmlGZipB64"` // The payload
}

// FetchDocuments retrieves a batch of documents starting from a specific NSU.
// The endpoint is typically /DFe/{NSU} or similar depending on the exact swagger spec.
func (c *Client) FetchDocuments(ctx context.Context, lastNSU int64) (*DocumentResponse, error) {
	// Format the endpoint path. Assuming /DFe/{NSU} based on previous research.
	path := fmt.Sprintf("/DFe/%d", lastNSU)

	var response DocumentResponse
	if err := c.Request(ctx, "GET", path, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
