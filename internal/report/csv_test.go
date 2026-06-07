package report

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestGenerateCSV(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "test.csv")

	issueDate := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	docs := []ReportRow{
		{
			CompanyRole:   nfse.CompanyRolePrestada,
			IssueDate:     issueDate,
			Competence:    "202606",
			ChaveAcesso:   "12345678901234567890123456789012345678901234",
			PrestadorCNPJ: "12345678000100",
			PrestadorName: "Prestador Teste",
			TomadorCNPJ:   "98765432000199",
			TomadorName:   "Tomador Teste",
			ServiceValue:  1500.50,
			ISSValue:      30.00,
			PISValue:      9.75,
			COFINSValue:   45.00,
			Status:        nfse.DocumentStatusNormal,
		},
	}

	err := GenerateCSV(docs, outPath)
	if err != nil {
		t.Fatalf("GenerateCSV failed: %v", err)
	}

	file, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open generated CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 rows (header + 1 doc), got %d", len(records))
	}

	header := records[0]
	if header[0] != "Competencia" {
		t.Errorf("expected header[0] to be Competencia, got %s", header[0])
	}

	row1 := records[1]
	// "Competencia", "Data Emissao", "Chave de Acesso", "Direcao", "Visibilidade",
	// "CNPJ Prestador", "Nome Prestador", "CNPJ Tomador", "Nome Tomador",
	// "Valor Servico", "ISS", "IRRF", "INSS", "PIS", "COFINS", "CSLL", "Status",
	if row1[0] != "202606" {
		t.Errorf("expected Competencia '202606', got %s", row1[0])
	}
	if row1[1] != "2026-06-07" {
		t.Errorf("expected IssueDate '2026-06-07', got %s", row1[1])
	}
	if row1[9] != "1500.50" {
		t.Errorf("expected Valor Servico '1500.50', got %s", row1[9])
	}
	if row1[13] != "9.75" {
		t.Errorf("expected PIS '9.75', got %s", row1[13])
	}
	if row1[16] != "normal" {
		t.Errorf("expected Status 'normal', got %s", row1[16])
	}
}
