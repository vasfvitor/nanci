package nfse

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// XMLDocument represents the basic structure of the NFS-e XML we care about right now.
type XMLDocument struct {
	XMLName xml.Name `xml:"NFSe"`
	InfNFSe struct {
		ChaveAcesso string `xml:"chNFSe"`
		DataEmissao string `xml:"dhEmi"`
		Competencia string `xml:"compNFSe"` // format usually YYYY-MM
		Prestador   struct {
			CNPJ string `xml:"CNPJ"`
			Nome string `xml:"xNome"`
		} `xml:"prest"`
		Tomador struct {
			CNPJ string `xml:"CNPJ"`
			Nome string `xml:"xNome"`
		} `xml:"toma"`
		Valores struct {
			ValorServico float64 `xml:"vServ"`
			// Other values will be added in Marco 4
		} `xml:"valores"`
	} `xml:"infNFSe"`
}

// DecodeXMLPayload decodes the base64-gzipped payload into raw XML bytes.
func DecodeXMLPayload(payloadBase64 string) ([]byte, string, error) {
	// 1. Decode Base64
	gzippedData, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// 2. Un-Gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(gzippedData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	xmlData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read uncompressed xml: %w", err)
	}

	// 3. Compute SHA-256 hash
	hash := sha256.Sum256(xmlData)
	hashHex := hex.EncodeToString(hash[:])

	return xmlData, hashHex, nil
}

// ParseXML extracts the basic information from the raw XML bytes.
func ParseXML(xmlData []byte, companyCNPJRoot string) (*Document, error) {
	var parsedXML XMLDocument
	if err := xml.Unmarshal(xmlData, &parsedXML); err != nil {
		return nil, fmt.Errorf("failed to parse xml: %w", err)
	}

	doc := &Document{
		ChaveAcesso:   parsedXML.InfNFSe.ChaveAcesso,
		PrestadorCNPJ: parsedXML.InfNFSe.Prestador.CNPJ,
		PrestadorName: parsedXML.InfNFSe.Prestador.Nome,
		TomadorCNPJ:   parsedXML.InfNFSe.Tomador.CNPJ,
		TomadorName:   parsedXML.InfNFSe.Tomador.Nome,
		ServiceValue:  parsedXML.InfNFSe.Valores.ValorServico,
		Status:        "normal",
	}

	// Parse date
	// dhEmi format usually is RFC3339 without timezone or with timezone: "2023-10-25T14:30:00-03:00"
	if parsedXML.InfNFSe.DataEmissao != "" {
		if t, err := time.Parse(time.RFC3339, parsedXML.InfNFSe.DataEmissao); err == nil {
			doc.IssueDate = t
		}
	}

	// Normalize Competence to YYYY-MM
	comp := parsedXML.InfNFSe.Competencia
	if len(comp) >= 7 {
		// Usually format is "2023-10-01" or "2023-10", we just need YYYY-MM
		doc.Competence = comp[:7]
	}

	// Determine Direction
	// If the company's CNPJ root matches the Prestador, it's "prestada"
	// If it matches Tomador, it's "tomada"
	// Else "intermediario"
	prestRoot := getRootSafely(doc.PrestadorCNPJ)
	tomRoot := getRootSafely(doc.TomadorCNPJ)

	if prestRoot == companyCNPJRoot {
		doc.Direction = "prestada"
	} else if tomRoot == companyCNPJRoot {
		doc.Direction = "tomada"
	} else {
		doc.Direction = "intermediario"
	}

	return doc, nil
}

func getRootSafely(cnpj string) string {
	cleaned := strings.ReplaceAll(cnpj, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) == 14 {
		return cleaned[:8]
	}
	return ""
}
