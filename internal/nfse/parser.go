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
		Intermediario struct {
			CNPJ string `xml:"CNPJ"`
			Nome string `xml:"xNome"`
		} `xml:"interm"`
		Valores struct {
			ValorServico float64 `xml:"vServ"`
			ISS          float64 `xml:"vISS"`
			IRRF         float64 `xml:"vIRRF"`
			INSS         float64 `xml:"vINSS"`
			PIS          float64 `xml:"vPIS"`
			COFINS       float64 `xml:"vCOFINS"`
			CSLL         float64 `xml:"vCSLL"`
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

// ParseXML extracts the canonical document information from the raw XML bytes.
func ParseXML(xmlData []byte) (*Document, error) {
	var parsedXML XMLDocument
	if err := xml.Unmarshal(xmlData, &parsedXML); err != nil {
		return nil, fmt.Errorf("failed to parse xml: %w", err)
	}

	doc := &Document{
		ChaveAcesso:       parsedXML.InfNFSe.ChaveAcesso,
		PrestadorCNPJ:     parsedXML.InfNFSe.Prestador.CNPJ,
		PrestadorName:     parsedXML.InfNFSe.Prestador.Nome,
		TomadorCNPJ:       parsedXML.InfNFSe.Tomador.CNPJ,
		TomadorName:       parsedXML.InfNFSe.Tomador.Nome,
		IntermediarioCNPJ: parsedXML.InfNFSe.Intermediario.CNPJ,
		IntermediarioName: parsedXML.InfNFSe.Intermediario.Nome,
		ServiceValue:      parsedXML.InfNFSe.Valores.ValorServico,
		ISSValue:          parsedXML.InfNFSe.Valores.ISS,
		IRRFValue:         parsedXML.InfNFSe.Valores.IRRF,
		INSSValue:         parsedXML.InfNFSe.Valores.INSS,
		PISValue:          parsedXML.InfNFSe.Valores.PIS,
		COFINSValue:       parsedXML.InfNFSe.Valores.COFINS,
		CSLLValue:         parsedXML.InfNFSe.Valores.CSLL,
		Status:            "normal",
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

	return doc, nil
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
