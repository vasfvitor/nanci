package adn

import (
	"context"
	"fmt"
	"net/url"
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

// DistributionRequest describes one contributor distribution query.
type DistributionRequest struct {
	LastNSU          int64
	ConsultationCNPJ string
}

// FetchDocuments retrieves a batch of documents starting from a specific NSU.
func (c *Client) FetchDocuments(ctx context.Context, req DistributionRequest) (*DocumentResponse, error) {
	path := fmt.Sprintf("/DFe/%d", req.LastNSU)
	if req.ConsultationCNPJ != "" {
		path = path + "?cnpjConsulta=" + url.QueryEscape(req.ConsultationCNPJ)
	}

	var response DocumentResponse
	// bodyProvider is nil for GET request
	if err := c.request(ctx, "GET", path, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
