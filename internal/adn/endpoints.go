package adn

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// DocumentResponse represents the API response for the GET /DFe endpoint.
type DocumentResponse struct {
	UltNSU int64              `json:"UltNSU"`
	MaxNSU int64              `json:"MaxNSU"`
	Docs   []DocumentEnvelope `json:"LoteDFe"`
}

// DocumentEnvelope wraps a single fiscal document.
type DocumentEnvelope struct {
	NSU           int64  `json:"NSU"`
	Schema        string `json:"Schema"`
	XMLGZipBase64 string `json:"ArquivoXml"`
	DocumentType  string `json:"TipoDocumento"`
	EventType     string `json:"TipoEvento"`
}

func (e DocumentEnvelope) PayloadBase64() string {
	return e.XMLGZipBase64
}

func (e DocumentEnvelope) IsEvent() bool {
	if e.EventType != "" {
		return true
	}
	if e.DocumentType != "" {
		return false
	}
	return strings.Contains(e.Schema, "procEvento")
}

// DistributionRequest describes one contributor distribution query.
type DistributionRequest struct {
	LastNSU          int64
	ConsultationCNPJ string
	Lote             *bool
}

// FetchDocuments retrieves a batch of documents starting from a specific NSU.
func (c *Client) FetchDocuments(ctx context.Context, req DistributionRequest) (*DocumentResponse, error) {
	rel := &url.URL{
		Path: fmt.Sprintf("DFe/%d", req.LastNSU),
	}
	if req.ConsultationCNPJ != "" {
		q := rel.Query()
		q.Set("cnpjConsulta", req.ConsultationCNPJ)
		if req.Lote != nil {
			q.Set("lote", fmt.Sprintf("%t", *req.Lote))
		}
		rel.RawQuery = q.Encode()
	} else if req.Lote != nil {
		q := rel.Query()
		q.Set("lote", fmt.Sprintf("%t", *req.Lote))
		rel.RawQuery = q.Encode()
	}
	path := rel.String()

	var response DocumentResponse
	// bodyProvider is nil for GET request
	if err := c.request(ctx, "GET", path, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
