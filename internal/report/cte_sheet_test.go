package report

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
)

const (
	cteSheetChaveProc = "35260912345678000195570010000001011123456784"
	cteSheetChaveOS   = "35260912345678000195670010000001041456789014"
)

func cteSheetDocs() []cte.CompanyDocument {
	issued := time.Date(2026, 9, 2, 8, 30, 0, 0, time.FixedZone("BRT", -3*3600))
	proc := cte.CompanyDocument{
		Document: cte.Document{
			ChaveAcesso:         cteSheetChaveProc,
			Modelo:              "57",
			TipoDocumento:       cte.TipoDocumentoCTe,
			Serie:               "1",
			Numero:              "101",
			CFOP:                "6353",
			NatOp:               "PRESTACAO DE SERVICO DE TRANSPORTE",
			IssueDate:           issued,
			Competence:          "2026-09",
			Protocolo:           "135260000000101",
			MunIni:              cte.Municipio{Nome: "SAO PAULO", UF: "SP"},
			MunFim:              cte.Municipio{Nome: "RIO DE JANEIRO", UF: "RJ"},
			Emitente:            cte.Party{CNPJ: "12345678000195", Name: "Transportadora"},
			Tomador:             cte.Party{CNPJ: "70860312000150", Name: "Empresa Mock"},
			Remetente:           cte.Party{CNPJ: "11222333000181", Name: "Remetente"},
			Destinatario:        cte.Party{CNPJ: "70860312000150", Name: "Empresa Mock"},
			TotalValue:          dfe.NewMoneyFromCents(150000),
			ReceivableValue:     dfe.NewMoneyFromCents(145000),
			ICMSValue:           dfe.NewMoneyFromCents(18000),
			TotTribValue:        dfe.NewMoneyFromCents(25035),
			CargaValue:          dfe.NewMoneyFromCents(2500000),
			ProdutoPredominante: "PECAS AUTOMOTIVAS",
			NFeChaves:           []string{"35260911222333000181550010000012341123456787", "35260911222333000181550010000012351234567893"},
			Situacao:            cte.SituacaoCancelada,
		},
		CompanyRole: cte.CompanyRoleTomador,
		Papeis:      []cte.CompanyRole{cte.CompanyRoleTomador, cte.CompanyRoleDestinatario},
		EventCount:  1,
	}
	// A CT-e OS carries no remetente, destinatário, carga or NF-e.
	osDoc := cte.CompanyDocument{
		Document: cte.Document{
			ChaveAcesso:   cteSheetChaveOS,
			Modelo:        "67",
			TipoDocumento: cte.TipoDocumentoCTeOS,
			Serie:         "1",
			Numero:        "104",
			CFOP:          "5357",
			NatOp:         "TRANSPORTE DE PESSOAS",
			IssueDate:     issued,
			Competence:    "2026-09",
			MunIni:        cte.Municipio{Nome: "SAO PAULO", UF: "SP"},
			MunFim:        cte.Municipio{UF: "SP"},
			Emitente:      cte.Party{CNPJ: "70860312000150", Name: "Empresa Mock"},
			Tomador:       cte.Party{CNPJ: "11223344000186", Name: "Cliente"},
			TotalValue:    dfe.NewMoneyFromCents(200000),
			Situacao:      cte.SituacaoAutorizada,
		},
		CompanyRole: cte.CompanyRoleEmitente,
		Papeis:      []cte.CompanyRole{cte.CompanyRoleEmitente},
	}
	other := cte.CompanyDocument{
		Document:    cte.Document{ChaveAcesso: "35260912345678000195570010000001021234567891", Modelo: "57", Situacao: cte.SituacaoAutorizada},
		CompanyRole: cte.CompanyRoleAutorizado,
		Papeis:      []cte.CompanyRole{cte.CompanyRoleAutorizado},
	}
	return []cte.CompanyDocument{other, osDoc, proc}
}

func TestCTeSpreadsheetRoutesRowsByRole(t *testing.T) {
	s := CTeSpreadsheet(BuildCTeRows(cteSheetDocs()))
	if err := s.validate(); err != nil {
		t.Fatal(err)
	}
	if s.Columns[s.StatusCol].Header != "Situação" {
		t.Errorf("StatusCol %d is %q, want Situação", s.StatusCol, s.Columns[s.StatusCol].Header)
	}

	chaveCol := slices.IndexFunc(s.Columns, func(c Column) bool { return c.Header == "Chave de Acesso" })
	got := map[string][]string{}
	for _, sheet := range s.Sheets {
		for _, row := range sheet.Rows {
			got[sheet.Name] = append(got[sheet.Name], row[chaveCol].(string))
		}
	}
	want := map[string][]string{
		"CTe Tomados":  {cteSheetChaveProc},
		"CTe Emitidos": {cteSheetChaveOS},
		"CTe Outros":   {"35260912345678000195570010000001021234567891"},
	}
	for name, chaves := range want {
		if !slices.Equal(got[name], chaves) {
			t.Errorf("sheet %s = %v, want %v", name, got[name], chaves)
		}
	}
}

func TestCTeSpreadsheetCSV(t *testing.T) {
	out := filepath.Join(t.TempDir(), "cte.csv")
	if err := WriteCSV(CTeSpreadsheet(BuildCTeRows(cteSheetDocs())), out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out) // #nosec G304 -- file created by this test.
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("csv has %d lines, want header and 3 rows:\n%s", len(lines), data)
	}

	header := "Papel,Papéis,Data Emissão,Competência,Modelo,Série,Número,Chave de Acesso,CFOP,Natureza da Operação," +
		"Início (município/UF),Fim (município/UF),CNPJ/CPF Emitente,Nome Emitente,CNPJ/CPF Tomador,Nome Tomador," +
		"CNPJ/CPF Remetente,Nome Remetente,CNPJ/CPF Destinatário,Nome Destinatário,Valor Prestação (R$)," +
		"Valor a Receber (R$),ICMS (R$),Tributos Aprox. (R$),Valor Carga (R$),Produto Predominante," +
		"NF-e Transportadas,Situação,Protocolo,Eventos"
	if lines[0] != header {
		t.Errorf("header =\n%s\nwant\n%s", lines[0], header)
	}

	// Sheets are written in order: tomados, emitidos, outros.
	proc := "tomador,\"tomador, destinatario\",2026-09-02,2026-09,57,1,101," + cteSheetChaveProc +
		",6353,PRESTACAO DE SERVICO DE TRANSPORTE,SAO PAULO/SP,RIO DE JANEIRO/RJ,12.345.678/0001-95,Transportadora," +
		"70.860.312/0001-50,Empresa Mock,11.222.333/0001-81,Remetente,70.860.312/0001-50,Empresa Mock," +
		"1500.00,1450.00,180.00,250.35,25000.00,PECAS AUTOMOTIVAS,2,cancelada,135260000000101,1"
	if lines[1] != proc {
		t.Errorf("procCTe row =\n%s\nwant\n%s", lines[1], proc)
	}
	cteOS := "emitente,emitente,2026-09-02,2026-09,67,1,104," + cteSheetChaveOS +
		",5357,TRANSPORTE DE PESSOAS,SAO PAULO/SP,SP,70.860.312/0001-50,Empresa Mock,11.223.344/0001-86,Cliente," +
		",,,,2000.00,0.00,0.00,0.00,0.00,,0,autorizada,,0"
	if lines[2] != cteOS {
		t.Errorf("CT-e OS row =\n%s\nwant\n%s", lines[2], cteOS)
	}
}

func TestCTeSpreadsheetXLSXSheets(t *testing.T) {
	out := filepath.Join(t.TempDir(), "cte.xlsx")
	if err := WriteXLSX(CTeSpreadsheet(BuildCTeRows(cteSheetDocs())), out); err != nil {
		t.Fatal(err)
	}
	f := openXLSX(t, out)
	if got, want := f.GetSheetList(), []string{"CTe Tomados", "CTe Emitidos", "CTe Outros"}; !slices.Equal(got, want) {
		t.Errorf("sheets = %v, want %v", got, want)
	}
	// Money cells hold numbers, not text.
	got, err := f.GetCellValue("CTe Tomados", "U2", excelize.Options{RawCellValue: true})
	if err != nil {
		t.Fatal(err)
	}
	if got != "1500" {
		t.Errorf("Valor Prestação raw = %q, want 1500", got)
	}
}
