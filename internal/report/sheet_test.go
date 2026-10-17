package report

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWriteCSVUsesCSVHeaderWhenSet(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.csv")
	s := Spreadsheet{
		Columns: []Column{
			{Header: "Situação", CSVHeader: "Situacao"},
			{Header: "Valor (R$)", CSVHeader: "Valor", Money: true},
			{Header: "Itens"},
			{Header: "Peso"},
		},
		StatusCol: -1,
		Sheets: []Sheet{
			{Name: "A", Rows: [][]any{{"normal", 10.5, 3, 1.25}}},
			{Name: "B"},
			{Name: "C", Rows: [][]any{{"cancelada", 0.0, 0, 2.0}}},
		},
	}
	if err := WriteCSV(s, outPath); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	got, err := os.ReadFile(outPath) // #nosec G304 -- file created by this test.
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	want := "Situacao,Valor,Itens,Peso\nnormal,10.50,3,1.25\ncancelada,0.00,0,2\n"
	if string(got) != want {
		t.Errorf("csv = %q, want %q", got, want)
	}
}

func TestWriteRejectsRowsOfTheWrongWidth(t *testing.T) {
	dir := t.TempDir()
	s := Spreadsheet{
		Columns:   []Column{{Header: "A"}, {Header: "B"}},
		StatusCol: -1,
		Sheets:    []Sheet{{Name: "S", Table: "STable", Rows: [][]any{{"only one"}}}},
	}
	if err := WriteCSV(s, filepath.Join(dir, "out.csv")); err == nil {
		t.Error("WriteCSV accepted a short row")
	}
	if err := WriteXLSX(s, filepath.Join(dir, "out.xlsx")); err == nil {
		t.Error("WriteXLSX accepted a short row")
	}
}

func TestWriteXLSXAllEmptyWritesDocumentosSheet(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.xlsx")
	s := Spreadsheet{
		Columns:   []Column{{Header: "A"}},
		StatusCol: 0,
		Sheets:    []Sheet{{Name: "Emitidas", Table: "EmitidasTable"}, {Name: "Tomadas", Table: "TomadasTable"}},
	}
	if err := WriteXLSX(s, outPath); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}
	f := openXLSX(t, outPath)
	if got := f.GetSheetList(); !slices.Equal(got, []string{"Documentos"}) {
		t.Errorf("sheets = %v, want [Documentos]", got)
	}
}

func TestWriteXLSXStylesMoneyColumns(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.xlsx")
	s := Spreadsheet{
		Columns:   []Column{{Header: "Nome", Center: true}, {Header: "Valor", Money: true, Width: 18}},
		StatusCol: -1,
		Sheets:    []Sheet{{Name: "Dados", Table: "DadosTable", Rows: [][]any{{"x", 1234.5}, {"y", 2.0}}}},
	}
	if err := WriteXLSX(s, outPath); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}
	f := openXLSX(t, outPath)

	for _, cell := range []string{"B2", "B3"} {
		styleID, err := f.GetCellStyle("Dados", cell)
		if err != nil {
			t.Fatalf("GetCellStyle %s: %v", cell, err)
		}
		style, err := f.GetStyle(styleID)
		if err != nil {
			t.Fatalf("GetStyle %s: %v", cell, err)
		}
		if style.NumFmt != 4 {
			t.Errorf("%s NumFmt = %d, want 4", cell, style.NumFmt)
		}
	}
	width, err := f.GetColWidth("Dados", "B")
	if err != nil {
		t.Fatalf("GetColWidth: %v", err)
	}
	if width != 18 {
		t.Errorf("width of B = %v, want 18", width)
	}
	tables, err := f.GetTables("Dados")
	if err != nil {
		t.Fatalf("GetTables: %v", err)
	}
	if len(tables) != 1 || tables[0].Name != "DadosTable" || tables[0].Range != "A1:B3" {
		t.Errorf("tables = %+v, want DadosTable over A1:B3", tables)
	}
}

func TestWriteXLSXStatusColumnPastZ(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "out.xlsx")
	const width = 30
	const statusCol = 27 // column AB
	columns := make([]Column, width)
	row := make([]any, width)
	for i := range columns {
		columns[i] = Column{Header: "C" + strings.Repeat("x", i)}
		row[i] = ""
	}
	row[statusCol] = "cancelada"
	s := Spreadsheet{
		Columns:   columns,
		StatusCol: statusCol,
		Sheets:    []Sheet{{Name: "Dados", Table: "DadosTable", Rows: [][]any{row}}},
	}
	if err := WriteXLSX(s, outPath); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}
	f := openXLSX(t, outPath)

	formats, err := f.GetConditionalFormats("Dados")
	if err != nil {
		t.Fatalf("GetConditionalFormats: %v", err)
	}
	rules, ok := formats["A2:AD2"]
	if !ok {
		t.Fatalf("no conditional format over A2:AD2, got %v", formats)
	}
	var criteria []string
	for _, rule := range rules {
		criteria = append(criteria, rule.Criteria)
	}
	want := []string{`=$AB2="cancelada"`, `=OR($AB2="substituida",$AB2="denegada")`}
	if !slices.Equal(criteria, want) {
		t.Errorf("criteria = %q, want %q", criteria, want)
	}
}

func openXLSX(t *testing.T, path string) *excelize.File {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
