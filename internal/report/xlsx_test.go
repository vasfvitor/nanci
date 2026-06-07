package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestGenerateXLSX(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "test.xlsx")

	issueDate := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	docs := []ReportRow{
		{
			CompanyRole:   nfse.CompanyRoleTomada,
			IssueDate:     issueDate,
			Competence:    "202606",
			ChaveAcesso:   "12345678901234567890123456789012345678901234",
			PrestadorCNPJ: "12345678000100",
			PrestadorName: "Prestador Teste",
			TomadorCNPJ:   "98765432000199",
			TomadorName:   "Tomador Teste",
			ServiceValue:  2500.00,
			ISSValue:      50.00,
			PISValue:      16.25,
			COFINSValue:   75.00,
			Status:        nfse.DocumentStatusNormal,
			Description:   "Consultoria em TI",
			WarningsCount: 1,
		},
	}

	err := GenerateXLSX(docs, outPath)
	if err != nil {
		t.Fatalf("GenerateXLSX failed: %v", err)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("XLSX file was not created")
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("XLSX file is empty")
	}
}
