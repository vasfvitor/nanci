package report

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfe"
)

const nfeCSVHeader = "Papel,Data Emissao,Competencia,Serie,Numero,Chave de Acesso,CNPJ/CPF Emitente,Nome Emitente,UF Emitente," +
	"CNPJ/CPF Destinatario,Nome Destinatario,Tipo,Natureza da Operacao,Finalidade,Valor Total,ICMS,IPI,Situacao,Completude," +
	"Manifestacao,Protocolo,Eventos"

func nfeSheetDoc(role nfe.CompanyRole, chave string) nfe.CompanyDocument {
	return nfe.CompanyDocument{
		Document: nfe.Document{
			ChaveAcesso:      dfe.AccessKey(chave),
			Serie:            "1",
			Numero:           "123",
			IssueDate:        time.Date(2026, 6, 7, 10, 0, 0, 0, time.UTC),
			Competence:       "2026-06",
			Protocolo:        "135260000000001",
			EmitenteCNPJ:     "12345678000100",
			EmitenteName:     "Emitente, Teste",
			EmitenteUF:       "SP",
			DestinatarioCNPJ: "98765432000199",
			DestinatarioName: "Destinatario",
			TpNF:             "1",
			FinNFe:           "4",
			NatOp:            "Devolucao de mercadoria",
			TotalValue:       dfe.NewMoneyFromCents(150050),
			ICMSValue:        dfe.NewMoneyFromCents(1805),
			IPIValue:         dfe.NewMoneyFromCents(1),
			Situacao:         nfe.SituacaoAutorizada,
			Completeness:     nfe.CompletenessCompleta,
		},
		CompanyRole:  role,
		Manifestacao: nfe.ManifestacaoConfirmada,
		EventCount:   2,
	}
}

func TestBuildNFeRowsConvertsCentsAndCodes(t *testing.T) {
	rows := BuildNFeRows([]nfe.CompanyDocument{nfeSheetDoc(nfe.CompanyRoleDestinatario, "A")})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	got := rows[0]
	if got.TotalValue != 1500.50 || got.ICMSValue != 18.05 || got.IPIValue != 0.01 {
		t.Errorf("values = %v, %v, %v; want 1500.50, 18.05, 0.01", got.TotalValue, got.ICMSValue, got.IPIValue)
	}
	if got.Tipo != "Saída" || got.Finalidade != "Devolução" || got.Manifestacao != "Confirmada" {
		t.Errorf("labels = %q, %q, %q", got.Tipo, got.Finalidade, got.Manifestacao)
	}
}

func TestNFeLabels(t *testing.T) {
	for code, want := range map[string]string{"0": "Entrada", "1": "Saída", "": "", "9": "9"} {
		if got := tipoLabel(code); got != want {
			t.Errorf("tipoLabel(%q) = %q, want %q", code, got, want)
		}
	}
	for code, want := range map[string]string{"1": "Normal", "2": "Complementar", "3": "Ajuste", "4": "Devolução", "": "", "7": "7"} {
		if got := finalidadeLabel(code); got != want {
			t.Errorf("finalidadeLabel(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestNFeSpreadsheetRoutesRowsBySheet(t *testing.T) {
	docs := []nfe.CompanyDocument{
		nfeSheetDoc(nfe.CompanyRoleDestinatario, "A"),
		nfeSheetDoc(nfe.CompanyRoleEmitente, "B"),
		nfeSheetDoc(nfe.CompanyRoleTransportador, "C"),
		nfeSheetDoc(nfe.CompanyRoleAutorizado, "D"),
		nfeSheetDoc(nfe.CompanyRoleNone, "E"),
		nfeSheetDoc(nfe.CompanyRoleDestinatario, "F"),
	}
	s := NFeSpreadsheet(BuildNFeRows(docs))

	chaves := func(sheet Sheet) []string {
		var out []string
		for _, row := range sheet.Rows {
			out = append(out, row[5].(string))
		}
		return out
	}
	want := map[string][]string{
		"NFe Recebidas": {"A", "F"},
		"NFe Emitidas":  {"B"},
		"NFe Outras":    {"C", "D", "E"},
	}
	if len(s.Sheets) != len(want) {
		t.Fatalf("sheets = %d, want %d", len(s.Sheets), len(want))
	}
	for _, sheet := range s.Sheets {
		if got := chaves(sheet); !slices.Equal(got, want[sheet.Name]) {
			t.Errorf("%s chaves = %v, want %v", sheet.Name, got, want[sheet.Name])
		}
	}
	if s.Columns[s.StatusCol].Header != "Situação" {
		t.Errorf("status column = %q, want Situação", s.Columns[s.StatusCol].Header)
	}
}

func TestNFeCSVOutput(t *testing.T) {
	resumo := nfe.CompanyDocument{
		Document: nfe.Document{
			ChaveAcesso:  "R",
			Serie:        "2",
			Numero:       "9",
			Competence:   "2026-06",
			EmitenteCNPJ: "12345678901",
			EmitenteName: "Emitente CPF",
			EmitenteUF:   "35",
			TpNF:         "0",
			TotalValue:   dfe.NewMoneyFromCents(1000),
			Situacao:     nfe.SituacaoCancelada,
			Completeness: nfe.CompletenessResumo,
		},
		CompanyRole:  nfe.CompanyRoleDestinatario,
		Manifestacao: nfe.ManifestacaoNenhuma,
	}
	outPath := filepath.Join(t.TempDir(), "nfe.csv")
	docs := []nfe.CompanyDocument{nfeSheetDoc(nfe.CompanyRoleEmitente, "A"), resumo}
	if err := WriteCSV(NFeSpreadsheet(BuildNFeRows(docs)), outPath); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	got, err := os.ReadFile(outPath) // #nosec G304 -- file created by this test.
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	// The CSV follows the sheet order: recebidas first, then emitidas.
	want := nfeCSVHeader + "\n" +
		"destinatario,,2026-06,2,9,R,12345678901,Emitente CPF,35,,,Entrada,,,10.00,0.00,0.00,cancelada,resumo,Sem manifestação,,0\n" +
		`emitente,2026-06-07,2026-06,1,123,A,12.345.678/0001-00,"Emitente, Teste",SP,98.765.432/0001-99,Destinatario,Saída,` +
		"Devolucao de mercadoria,Devolução,1500.50,18.05,0.01,autorizada,completa,Confirmada,135260000000001,2\n"
	if string(got) != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
	if strings.ContainsFunc(nfeCSVHeader, func(r rune) bool { return r > 127 }) {
		t.Errorf("CSV header has non-ASCII characters: %q", nfeCSVHeader)
	}
}

func TestNFeXLSXWritesOneSheetPerRoleWithRows(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "nfe.xlsx")
	docs := []nfe.CompanyDocument{
		nfeSheetDoc(nfe.CompanyRoleDestinatario, "A"),
		nfeSheetDoc(nfe.CompanyRoleNone, "B"),
	}
	if err := WriteXLSX(NFeSpreadsheet(BuildNFeRows(docs)), outPath); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}
	f := openXLSX(t, outPath)
	if got := f.GetSheetList(); !slices.Equal(got, []string{"NFe Recebidas", "NFe Outras"}) {
		t.Errorf("sheets = %v, want [NFe Recebidas NFe Outras]", got)
	}
	header, err := f.GetCellValue("NFe Recebidas", "R1")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if header != "Situação" {
		t.Errorf("R1 = %q, want Situação", header)
	}
}
