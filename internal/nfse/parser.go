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
	"strconv"
	"strings"
	"time"
)

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

// ParseXML extracts the canonical document information from the raw XML bytes using a tolerant tag parser.
func ParseXML(xmlData []byte) (*Document, error) {
	doc := &Document{
		Status: "normal",
	}

	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	
	var contextStack []string
	
	inContext := func(parent string) bool {
		if len(contextStack) == 0 {
			return false
		}
		// Look back for the parent since there might be nested tags
		for i := len(contextStack) - 1; i >= 0; i-- {
			if contextStack[i] == parent {
				return true
			}
		}
		return false
	}

	var currentText string

	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse xml: %w", err)
		}

		switch se := t.(type) {
		case xml.StartElement:
			local := se.Name.Local
			contextStack = append(contextStack, local)
			currentText = "" // reset text

			if local == "infNFSe" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "versao" {
						doc.LayoutVersion = attr.Value
					}
				}
			}

		case xml.EndElement:
			local := se.Name.Local
			
			// Pop context
			if len(contextStack) > 0 {
				contextStack = contextStack[:len(contextStack)-1]
			}
			
			val := strings.TrimSpace(currentText)
			currentText = "" // clear after reading
			
			if val == "" {
				continue
			}

			switch local {
			case "chNFSe":
				doc.ChaveAcesso = val
			case "dhEmi":
				if parsed, err := time.Parse(time.RFC3339, val); err == nil {
					doc.IssueDate = parsed
				} else {
					doc.ParseWarnings = append(doc.ParseWarnings, fmt.Sprintf("invalid dhEmi format: %s", val))
				}
			case "compNFSe":
				if len(val) >= 7 {
					doc.Competence = val[:7]
				} else {
					doc.Competence = val
				}
			case "CNPJ":
				if inContext("prest") {
					doc.PrestadorCNPJ = val
				} else if inContext("toma") {
					doc.TomadorCNPJ = val
				} else if inContext("interm") {
					doc.IntermediarioCNPJ = val
				}
			case "xNome":
				if inContext("prest") {
					doc.PrestadorName = val
				} else if inContext("toma") {
					doc.TomadorName = val
				} else if inContext("interm") {
					doc.IntermediarioName = val
				}
			case "vServ":
				doc.ServiceValue = parseFloat(val)
			case "vISS":
				doc.ISSValue = parseFloat(val)
			case "vIRRF":
				doc.IRRFValue = parseFloat(val)
			case "vINSS":
				doc.INSSValue = parseFloat(val)
			case "vPIS":
				doc.PISValue = parseFloat(val)
			case "vCOFINS":
				doc.COFINSValue = parseFloat(val)
			case "vCSLL":
				doc.CSLLValue = parseFloat(val)
			case "nNFSe", "numero":
				if doc.NFSeNumber == "" {
					doc.NFSeNumber = val
				}
			case "xDesc", "descTribNac", "Discriminacao", "discriminacao", "Descricao", "descricao":
				if doc.ServiceDescription == "" {
					if len(val) > 200 {
						doc.ServiceDescription = val[:197] + "..."
					} else {
						doc.ServiceDescription = val
					}
				}
			}

		case xml.CharData:
			currentText += string(se)
		}
	}

	if doc.ChaveAcesso == "" {
		return nil, fmt.Errorf("missing chave de acesso (chNFSe)")
	}

	doc.TotalRetentions = doc.IRRFValue + doc.INSSValue + doc.PISValue + doc.COFINSValue + doc.CSLLValue
	if doc.ISSValue > 0 {
		doc.ParseWarnings = append(doc.ParseWarnings, "ISS presente, mas nanci ainda não distingue ISS devido de retido")
	}

	if doc.Competence == "" {
		doc.ParseWarnings = append(doc.ParseWarnings, "competência ausente")
	}

	if doc.PrestadorCNPJ == "" || doc.TomadorCNPJ == "" {
		doc.ParseWarnings = append(doc.ParseWarnings, "prestador ou tomador ausente")
	}

	return doc, nil
}

func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ClassifyCompanyParticipation derives company-scoped role and visibility for a canonical document.
func ClassifyCompanyParticipation(doc *Document, companyCNPJ string) CompanyParticipation {
	companyCNPJ = normalizeCNPJ(companyCNPJ)

	switch companyCNPJ {
	case normalizeCNPJ(doc.PrestadorCNPJ):
		return CompanyParticipation{CompanyRole: "prestada", VisibilityReason: "exact_prestador"}
	case normalizeCNPJ(doc.TomadorCNPJ):
		return CompanyParticipation{CompanyRole: "tomada", VisibilityReason: "exact_tomador"}
	case normalizeCNPJ(doc.IntermediarioCNPJ):
		return CompanyParticipation{CompanyRole: "intermediario", VisibilityReason: "exact_intermediario"}
	}

	companyRoot := getRootSafely(companyCNPJ)
	if companyRoot != "" && (companyRoot == getRootSafely(doc.PrestadorCNPJ) ||
		companyRoot == getRootSafely(doc.TomadorCNPJ) ||
		companyRoot == getRootSafely(doc.IntermediarioCNPJ)) {
		return CompanyParticipation{CompanyRole: "none", VisibilityReason: "same_root_only"}
	}

	return CompanyParticipation{CompanyRole: "none", VisibilityReason: "unknown"}
}

// ParseEvent extracts basic info from an Event XML.
func ParseEvent(xmlData []byte) (*Event, error) {
	strData := string(xmlData)
	chaveAcesso := firstTagValue(strData, "chNFSe")
	if chaveAcesso == "" {
		return nil, fmt.Errorf("could not find chNFSe in event")
	}

	event := &Event{
		ChaveAcesso: chaveAcesso,
		Type:        classifyEventType(strData),
		Description: "Evento sincronizado via NSU",
	}

	if ts := firstTagValue(strData, "dhEvento", "dhRegEvento", "dhProc", "dtHrEvento"); ts != "" {
		if parsedTime, err := time.Parse(time.RFC3339, ts); err == nil {
			event.EventAt = parsedTime
			event.EventAtValid = true
		}
	}

	event.ReplacementChaveAcesso = firstTagValue(
		strData,
		"chNFSeSubst",
		"chNFSeSubstituta",
		"chNFSeSubstituidora",
		"chNFSeSubstituidaPor",
	)
	if event.ReplacementChaveAcesso != "" && event.Type == "unknown" {
		event.Type = "substituicao"
	}

	if reason := firstTagValue(strData, "xMotivo", "cMotivo", "descEvento", "xJust"); reason != "" {
		event.Description = reason
	}

	return event, nil
}

func getRootSafely(cnpj string) string {
	cleaned := normalizeCNPJ(cnpj)
	if len(cleaned) == 14 {
		return cleaned[:8]
	}
	return ""
}

func normalizeCNPJ(cnpj string) string {
	cleaned := strings.ReplaceAll(cnpj, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	return strings.TrimSpace(cleaned)
}

func classifyEventType(xmlText string) string {
	lower := strings.ToLower(xmlText)
	switch {
	case strings.Contains(lower, "substitu"):
		return "substituicao"
	case strings.Contains(lower, "canc"):
		return "cancelamento"
	default:
		return "unknown"
	}
}

func firstTagValue(xmlText string, tags ...string) string {
	for _, tag := range tags {
		openTag := "<" + tag + ">"
		start := strings.Index(xmlText, openTag)
		if start == -1 {
			continue
		}
		contentStart := start + len(openTag)
		end := strings.Index(xmlText[contentStart:], "</"+tag+">")
		if end == -1 {
			continue
		}
		value := strings.TrimSpace(xmlText[contentStart : contentStart+end])
		if value != "" {
			return value
		}
	}
	return ""
}
