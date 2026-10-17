package report

import (
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// NFSeCSV lays out the NFS-e CSV export: one row per document, in order,
// under ASCII headers.
func NFSeCSV(rows []NFSeRow) Spreadsheet {
	columns := []Column{
		{Header: "Competencia"},
		{Header: "Data Emissao"},
		{Header: "Chave de Acesso"},
		{Header: "Direcao"},
		{Header: "CNPJ Prestador"},
		{Header: "Nome Prestador"},
		{Header: "CNPJ Tomador"},
		{Header: "Nome Tomador"},
		{Header: "Valor Servico", Money: true},
		{Header: "ISS", Money: true},
		{Header: "IRRF", Money: true},
		{Header: "INSS", Money: true},
		{Header: "PIS", Money: true},
		{Header: "COFINS", Money: true},
		{Header: "CSLL", Money: true},
		{Header: "Status"},
	}

	cells := make([][]any, 0, len(rows))
	for _, doc := range rows {
		cells = append(cells, []any{
			doc.Competence,
			formatIssueDate(doc.IssueDate),
			doc.ChaveAcesso,
			string(doc.CompanyRole),
			cnpj.Format(doc.PrestadorCNPJ),
			doc.PrestadorName,
			cnpj.Format(doc.TomadorCNPJ),
			doc.TomadorName,
			doc.ServiceValue,
			doc.ISSValue,
			doc.IRRFValue,
			doc.INSSValue,
			doc.PISValue,
			doc.COFINSValue,
			doc.CSLLValue,
			string(doc.Status),
		})
	}

	return Spreadsheet{
		Columns:   columns,
		StatusCol: 15,
		Sheets:    []Sheet{{Name: "Documentos", Table: "DocumentosTable", Rows: cells}},
	}
}

// formatIssueDate is the issue date as YYYY-MM-DD, or empty when unknown.
func formatIssueDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.DateOnly)
}
