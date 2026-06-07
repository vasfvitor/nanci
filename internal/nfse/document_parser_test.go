package nfse

import (
	"strings"
	"testing"
	"time"
)

func TestParseDocumentXML_Valid(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<NFSe xmlns="http://www.sped.fazenda.gov.br/nfse">
  <infNFSe versao="1.00">
    <chNFSe>12345678901234567890123456789012345678901234567890</chNFSe>
    <nNFSe>10001</nNFSe>
    <dhEmi>2026-06-07T10:00:00-03:00</dhEmi>
    <compNFSe>2026-06</compNFSe>
    <prest>
      <CNPJ>12345678000100</CNPJ>
      <xNome>Prestador Teste</xNome>
    </prest>
    <toma>
      <CNPJ>98765432000199</CNPJ>
      <xNome>Tomador Teste</xNome>
    </toma>
    <valores>
      <vServ>1500.50</vServ>
      <vISS>30.00</vISS>
      <vIRRF>10.00</vIRRF>
      <vINSS>5.00</vINSS>
      <vPIS>9.75</vPIS>
      <vCOFINS>45.00</vCOFINS>
      <vCSLL>15.00</vCSLL>
    </valores>
    <xDescServ>Consultoria e desenvolvimento</xDescServ>
  </infNFSe>
</NFSe>`

	doc, warnings, err := ParseDocumentXML([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseDocumentXML failed: %v", err)
	}

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}

	if doc.LayoutVersion != "1.00" {
		t.Errorf("expected versao 1.00, got %s", doc.LayoutVersion)
	}

	if doc.ChaveAcesso != "12345678901234567890123456789012345678901234567890" {
		t.Errorf("expected chave 12345678901234567890123456789012345678901234567890, got %s", doc.ChaveAcesso)
	}

	if doc.NFSeNumber != "10001" {
		t.Errorf("expected NFSeNumber 10001, got %s", doc.NFSeNumber)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2026-06-07T10:00:00-03:00")
	if !doc.IssueDate.Equal(expectedTime) {
		t.Errorf("expected IssueDate %v, got %v", expectedTime, doc.IssueDate)
	}

	if doc.Competence != "2026-06" {
		t.Errorf("expected Competence 2026-06, got %s", doc.Competence)
	}

	if doc.PrestadorCNPJ != "12345678000100" {
		t.Errorf("expected PrestadorCNPJ 12345678000100, got %s", doc.PrestadorCNPJ)
	}

	if doc.TomadorName != "Tomador Teste" {
		t.Errorf("expected TomadorName 'Tomador Teste', got %s", doc.TomadorName)
	}

	if doc.ServiceValue != 150050 {
		t.Errorf("expected ServiceValue 150050, got %d", doc.ServiceValue)
	}

	if doc.ISSValue != 3000 {
		t.Errorf("expected ISSValue 3000, got %d", doc.ISSValue)
	}

	if doc.TotalRetentions != 8475 { // 10.00 + 5.00 + 9.75 + 45.00 + 15.00 = 84.75 -> 8475
		t.Errorf("expected TotalRetentions 8475, got %d", doc.TotalRetentions)
	}

	if doc.ServiceDescription != "Consultoria e desenvolvimento" {
		t.Errorf("expected ServiceDescription 'Consultoria e desenvolvimento', got %s", doc.ServiceDescription)
	}
}

func TestParseDocumentXML_InvalidOrMissingChave(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?><NFSe></NFSe>`
	_, _, err := ParseDocumentXML([]byte(xmlData))
	if err == nil {
		t.Fatal("expected error for missing chave, got nil")
	}

	if !strings.Contains(err.Error(), "missing essential field: chNFSe") {
		t.Errorf("expected missing field error, got %v", err)
	}
}

func TestParseDocumentXML_Empty(t *testing.T) {
	_, _, err := ParseDocumentXML([]byte("   "))
	if err == nil {
		t.Fatal("expected error for empty XML, got nil")
	}
	if !strings.Contains(err.Error(), "empty xml document") {
		t.Errorf("expected empty xml error, got %v", err)
	}
}
