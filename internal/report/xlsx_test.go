package report

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestNFSeXLSXPlacesEveryRole(t *testing.T) {
	rows := BuildNFSeRows([]nfse.CompanyDocument{
		{Document: nfse.Document{ChaveAcesso: "P1"}, CompanyRole: nfse.CompanyRolePrestada},
		{Document: nfse.Document{ChaveAcesso: "T1"}, CompanyRole: nfse.CompanyRoleTomada},
		{Document: nfse.Document{ChaveAcesso: "T2"}, CompanyRole: nfse.CompanyRoleTomada},
		{Document: nfse.Document{ChaveAcesso: "I1"}, CompanyRole: nfse.CompanyRoleIntermediario},
		{Document: nfse.Document{ChaveAcesso: "N1"}, CompanyRole: "none"},
		{Document: nfse.Document{ChaveAcesso: "E1"}, CompanyRole: ""},
	})

	s := NFSeXLSX(rows)
	got := map[string]int{}
	total := 0
	for _, sheet := range s.Sheets {
		got[sheet.Name] = len(sheet.Rows)
		total += len(sheet.Rows)
	}
	want := map[string]int{"NFSe Emitidas": 1, "NFSe Tomadas": 2, "NFSe Intermediadas": 1, "NFSe Outras": 2}
	for name, n := range want {
		if got[name] != n {
			t.Errorf("sheet %s has %d rows, want %d", name, got[name], n)
		}
	}
	if total != len(rows) {
		t.Errorf("sheets hold %d rows, want %d", total, len(rows))
	}
}

func TestNFSeXLSXWritesOneWorksheetPerRole(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "test.xlsx")
	rows := BuildNFSeRows([]nfse.CompanyDocument{
		{
			Document: nfse.Document{
				ChaveAcesso:  "12345678901234567890123456789012345678901234",
				Competence:   "202606",
				ServiceValue: 250000,
				Status:       nfse.DocumentStatusNormal,
			},
			CompanyRole: nfse.CompanyRoleTomada,
		},
		{Document: nfse.Document{ChaveAcesso: "I1"}, CompanyRole: nfse.CompanyRoleIntermediario},
	})

	if err := WriteXLSX(NFSeXLSX(rows), outPath); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}

	f, err := excelize.OpenFile(outPath)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer func() { _ = f.Close() }()

	if got, want := f.GetSheetList(), []string{"NFSe Tomadas", "NFSe Intermediadas"}; !slices.Equal(got, want) {
		t.Fatalf("sheets = %v, want %v", got, want)
	}
	value, err := f.GetCellValue("NFSe Tomadas", "G2")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if value != "2,500.00" {
		t.Errorf("Valor Serviço = %q, want 2,500.00", value)
	}
}
