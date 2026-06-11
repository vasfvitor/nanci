package adn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// DocumentResponse represents the API response for the GET /DFe endpoint.
type DocumentResponse struct {
	UltNSU int64              `json:"ultNSU"`
	MaxNSU int64              `json:"maxNSU"`
	Docs   []DocumentEnvelope `json:"docFisc"`
}

type documentResponseWire struct {
	UltNSU int64              `json:"ultNSU"`
	MaxNSU int64              `json:"maxNSU"`
	Docs   []DocumentEnvelope `json:"docFisc"`
}

type documentResponseWireOfficial struct {
	UltNSU int64              `json:"UltNSU"`
	MaxNSU int64              `json:"MaxNSU"`
	Docs   []DocumentEnvelope `json:"LoteDFe"`
}

func (r *DocumentResponse) UnmarshalJSON(data []byte) error {
	var legacy documentResponseWire
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}

	var official documentResponseWireOfficial
	if err := json.Unmarshal(data, &official); err != nil {
		return err
	}

	r.UltNSU = firstNonZeroInt64(official.UltNSU, legacy.UltNSU)
	r.MaxNSU = firstNonZeroInt64(official.MaxNSU, legacy.MaxNSU)
	if len(official.Docs) > 0 {
		r.Docs = official.Docs
	} else {
		r.Docs = legacy.Docs
	}
	return nil
}

// DocumentEnvelope wraps a single fiscal document.
type DocumentEnvelope struct {
	NSU           int64  `json:"nsu"`
	Schema        string `json:"schema"`         // e.g., "procNFSe_v1.00.xsd"
	XMLGZipBase64 string `json:"nfseXmlGZipB64"` // Legacy payload field kept for compatibility.
	DocumentType  string `json:"tipoDocumento"`
	EventType     string `json:"tipoEvento"`
}

type documentEnvelopeWire struct {
	NSU           int64  `json:"nsu"`
	Schema        string `json:"schema"`
	XMLGZipBase64 string `json:"nfseXmlGZipB64"`
	DocumentType  string `json:"tipoDocumento"`
	EventType     string `json:"tipoEvento"`
}

type documentEnvelopeWireOfficial struct {
	NSU           int64  `json:"NSU"`
	Schema        string `json:"Schema"`
	XMLGZipBase64 string `json:"ArquivoXml"`
	DocumentType  string `json:"TipoDocumento"`
	EventType     string `json:"TipoEvento"`
}

func (e *DocumentEnvelope) UnmarshalJSON(data []byte) error {
	var legacy documentEnvelopeWire
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}

	var official documentEnvelopeWireOfficial
	if err := json.Unmarshal(data, &official); err != nil {
		return err
	}

	e.NSU = firstNonZeroInt64(official.NSU, legacy.NSU)
	e.Schema = firstNonEmpty(official.Schema, legacy.Schema)
	e.XMLGZipBase64 = firstNonEmpty(official.XMLGZipBase64, legacy.XMLGZipBase64)
	e.DocumentType = firstNonEmpty(official.DocumentType, legacy.DocumentType)
	e.EventType = firstNonEmpty(official.EventType, legacy.EventType)
	return nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
