package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// TestNFSeCSVOutput pins the NFS-e CSV byte for byte: header line, quoting,
// CNPJ formatting, two-decimal money and empty issue dates. The expected
// bytes were produced by the CSV writer this one replaced.
func TestNFSeCSVOutput(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "test.csv")
	rows := []NFSeRow{
		{
			CompanyRole:   nfse.CompanyRolePrestada,
			IssueDate:     time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC),
			Competence:    "202606",
			ChaveAcesso:   "12345678901234567890123456789012345678901234",
			PrestadorCNPJ: "12345678000100",
			PrestadorName: "Prestador, Teste",
			TomadorCNPJ:   "98765432000199",
			TomadorName:   `Tomador "Teste"`,
			ServiceValue:  1500.50,
			ISSValue:      30.00,
			PISValue:      9.75,
			COFINSValue:   45.00,
			Status:        nfse.DocumentStatusNormal,
		},
		{
			CompanyRole: nfse.CompanyRoleIntermediario,
			Competence:  "202606",
			ChaveAcesso: "B",
			TomadorCNPJ: "12345678901",
			IRRFValue:   0.005,
			Status:      nfse.DocumentStatusCancelada,
		},
	}

	if err := WriteCSV(NFSeCSV(rows), outPath); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	got, err := os.ReadFile(outPath) // #nosec G304 -- file created by this test.
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	want := "Competencia,Data Emissao,Chave de Acesso,Direcao,CNPJ Prestador,Nome Prestador,CNPJ Tomador,Nome Tomador,Valor Servico,ISS,IRRF,INSS,PIS,COFINS,CSLL,Status\n" +
		`202606,2026-06-07,12345678901234567890123456789012345678901234,prestada,12.345.678/0001-00,"Prestador, Teste",98.765.432/0001-99,"Tomador ""Teste""",1500.50,30.00,0.00,0.00,9.75,45.00,0.00,normal` + "\n" +
		"202606,,B,intermediario,,,12345678901,,0.00,0.00,0.01,0.00,0.00,0.00,0.00,cancelada\n"
	if string(got) != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
}

func TestNFSeCSVEmptyWritesHeaderOnly(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "empty.csv")
	if err := WriteCSV(NFSeCSV(nil), outPath); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	got, err := os.ReadFile(outPath) // #nosec G304 -- file created by this test.
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	want := "Competencia,Data Emissao,Chave de Acesso,Direcao,CNPJ Prestador,Nome Prestador,CNPJ Tomador,Nome Tomador,Valor Servico,ISS,IRRF,INSS,PIS,COFINS,CSLL,Status\n"
	if string(got) != want {
		t.Errorf("csv = %q, want %q", got, want)
	}
}
