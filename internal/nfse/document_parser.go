package nfse

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ParseDocumentXML parses a National NFS-e XML document using strict, namespace-aware path tracking.
func ParseDocumentXML(data []byte) (Document, []string, error) {
	var doc Document
	var warnings []string

	doc.Status = DocumentStatus("normal") // Assume normal unless event changes it
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt

	decoder := xml.NewDecoder(bytes.NewReader(data))

	// Fast track to reject obviously invalid XML before reading tokens
	if len(bytes.TrimSpace(data)) == 0 {
		return doc, nil, errors.New("empty xml document")
	}

	var pathStack []string
	var currentText strings.Builder

	for {
		t, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return doc, nil, fmt.Errorf("xml parse error: %w", err)
		}

		switch se := t.(type) {
		case xml.StartElement:
			// Just use the local name for path tracking to gracefully handle differing namespace prefixes
			local := se.Name.Local
			pathStack = append(pathStack, local)
			currentText.Reset()

			if local == "infNFSe" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "versao" {
						doc.LayoutVersion = attr.Value
					}
				}
			}

		case xml.CharData:
			currentText.Write(se)

		case xml.EndElement:
			val := strings.TrimSpace(currentText.String())
			currentPath := strings.Join(pathStack, "/")

			// Pop stack
			if len(pathStack) > 0 {
				pathStack = pathStack[:len(pathStack)-1]
			}

			if val == "" {
				continue
			}

			// We match specific paths starting with either NFS-e or generic tags since
			// the wrapper tags might vary depending on whether this came from an event or a direct fetch.
			// To be resilient to wrappers, we check suffixes.

			//nolint:gocritic // A flat suffix dispatch keeps XML field mappings readable.
			if strings.HasSuffix(currentPath, "/chNFSe") {
				parsed, err := ParseAccessKey(val)
				if err != nil {
					return doc, nil, fmt.Errorf("invalid document access key: %w", err)
				}
				doc.ChaveAcesso = parsed
			} else if strings.HasSuffix(currentPath, "/dhEmi") {
				if parsed, err := time.Parse(time.RFC3339, val); err == nil {
					doc.IssueDate = parsed
				} else {
					warnings = append(warnings, fmt.Sprintf("invalid dhEmi format: %s", val))
				}
			} else if strings.HasSuffix(currentPath, "/compNFSe") {
				if len(val) >= 7 {
					doc.Competence = val[:7]
				} else {
					doc.Competence = val
				}
			} else if strings.HasSuffix(currentPath, "/nNFSe") {
				doc.NFSeNumber = val
			} else if strings.HasSuffix(currentPath, "/prest/CNPJ") {
				doc.PrestadorCNPJ = val
			} else if strings.HasSuffix(currentPath, "/prest/xNome") {
				doc.PrestadorName = val
			} else if strings.HasSuffix(currentPath, "/toma/CNPJ") || strings.HasSuffix(currentPath, "/toma/CPF") || strings.HasSuffix(currentPath, "/toma/NIF") {
				doc.TomadorCNPJ = val
			} else if strings.HasSuffix(currentPath, "/toma/xNome") {
				doc.TomadorName = val
			} else if strings.HasSuffix(currentPath, "/interm/CNPJ") || strings.HasSuffix(currentPath, "/interm/CPF") { //nolint:misspell // Official XML tag.
				doc.IntermediarioCNPJ = val
			} else if strings.HasSuffix(currentPath, "/interm/xNome") { //nolint:misspell // Official XML tag.
				doc.IntermediarioName = val
			} else if strings.HasSuffix(currentPath, "/vServ") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vServ: %w", err)
				}
				doc.ServiceValue = m
			} else if strings.HasSuffix(currentPath, "/vISS") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vISS: %w", err)
				}
				doc.ISSValue = m
			} else if strings.HasSuffix(currentPath, "/vIRRF") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vIRRF: %w", err)
				}
				doc.IRRFValue = m
			} else if strings.HasSuffix(currentPath, "/vINSS") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vINSS: %w", err)
				}
				doc.INSSValue = m
			} else if strings.HasSuffix(currentPath, "/vPIS") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vPIS: %w", err)
				}
				doc.PISValue = m
			} else if strings.HasSuffix(currentPath, "/vCOFINS") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vCOFINS: %w", err)
				}
				doc.COFINSValue = m
			} else if strings.HasSuffix(currentPath, "/vCSLL") {
				m, err := ParseMoney(val)
				if err != nil {
					return doc, nil, fmt.Errorf("vCSLL: %w", err)
				}
				doc.CSLLValue = m
			} else if strings.HasSuffix(currentPath, "/xDescServ") {
				if doc.ServiceDescription != "" {
					doc.ServiceDescription += " " + val
				} else {
					doc.ServiceDescription = val
				}
			}

			// Reset text builder for next element
			currentText.Reset()
		}
	}

	// Validate essential fields
	if doc.ChaveAcesso == "" {
		return doc, nil, errors.New("missing essential field: chNFSe")
	}

	doc.ParseWarnings = warnings
	doc.TotalRetentions = doc.IRRFValue + doc.INSSValue + doc.PISValue + doc.COFINSValue + doc.CSLLValue
	return doc, warnings, nil
}
