package report

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
)

// NFeRow is one NF-e in the spreadsheet exports, with values in reais and
// the codes already turned into labels. Fields a resumo does not carry stay
// empty or zero.
type NFeRow struct {
	CompanyRole      nfe.CompanyRole
	IssueDate        time.Time
	Competence       string
	Serie            string
	Numero           string
	ChaveAcesso      string
	EmitenteCNPJ     string
	EmitenteName     string
	EmitenteUF       string
	DestinatarioCNPJ string
	DestinatarioName string
	Tipo             string // Entrada or Saída
	NatOp            string
	Finalidade       string
	TotalValue       float64
	ICMSValue        float64
	IPIValue         float64
	Situacao         nfe.Situacao
	Completeness     nfe.Completeness
	Manifestacao     string
	Protocolo        string
	EventCount       int
}

// BuildNFeRows turns company documents into spreadsheet rows, in order.
func BuildNFeRows(docs []nfe.CompanyDocument) []NFeRow {
	rows := make([]NFeRow, 0, len(docs))
	for _, doc := range docs {
		rows = append(rows, NFeRow{
			CompanyRole:      doc.CompanyRole,
			IssueDate:        doc.IssueDate,
			Competence:       doc.Competence,
			Serie:            doc.Serie,
			Numero:           doc.Numero,
			ChaveAcesso:      string(doc.ChaveAcesso),
			EmitenteCNPJ:     doc.EmitenteCNPJ,
			EmitenteName:     doc.EmitenteName,
			EmitenteUF:       doc.EmitenteUF,
			DestinatarioCNPJ: doc.DestinatarioCNPJ,
			DestinatarioName: doc.DestinatarioName,
			Tipo:             tipoLabel(doc.TpNF),
			NatOp:            doc.NatOp,
			Finalidade:       finalidadeLabel(doc.FinNFe),
			TotalValue:       reais(doc.TotalValue),
			ICMSValue:        reais(doc.ICMSValue),
			IPIValue:         reais(doc.IPIValue),
			Situacao:         doc.Situacao,
			Completeness:     doc.Completeness,
			Manifestacao:     doc.Manifestacao.Label(),
			Protocolo:        doc.Protocolo,
			EventCount:       doc.EventCount,
		})
	}
	return rows
}

// NFeSpreadsheet lays out the NF-e CSV and Excel exports: one sheet for the
// NF-e the company received, one for those it issued and one for every other
// role, so every row lands somewhere.
func NFeSpreadsheet(rows []NFeRow) Spreadsheet {
	columns := []Column{
		{Header: "Papel", Width: 14, Center: true},
		{Header: "Data Emissão", CSVHeader: "Data Emissao", Width: 12, Center: true},
		{Header: "Competência", CSVHeader: "Competencia", Width: 12, Center: true},
		{Header: "Série", CSVHeader: "Serie", Width: 8, Center: true},
		{Header: "Número", CSVHeader: "Numero", Width: 12, Center: true},
		{Header: "Chave de Acesso", Width: 47},
		{Header: "CNPJ/CPF Emitente", Width: 20, Center: true},
		{Header: "Nome Emitente", Width: 40},
		{Header: "UF Emitente", Width: 8, Center: true},
		{Header: "CNPJ/CPF Destinatário", CSVHeader: "CNPJ/CPF Destinatario", Width: 20, Center: true},
		{Header: "Nome Destinatário", CSVHeader: "Nome Destinatario", Width: 40},
		{Header: "Tipo", Width: 10, Center: true},
		{Header: "Natureza da Operação", CSVHeader: "Natureza da Operacao", Width: 30},
		{Header: "Finalidade", Width: 14, Center: true},
		{Header: "Valor Total (R$)", CSVHeader: "Valor Total", Width: 18, Money: true},
		{Header: "ICMS (R$)", CSVHeader: "ICMS", Width: 18, Money: true},
		{Header: "IPI (R$)", CSVHeader: "IPI", Width: 18, Money: true},
		{Header: "Situação", CSVHeader: "Situacao", Width: 12, Center: true},
		{Header: "Completude", Width: 12, Center: true},
		{Header: "Manifestação", CSVHeader: "Manifestacao", Width: 22, Center: true},
		{Header: "Protocolo", Width: 18, Center: true},
		{Header: "Eventos", Width: 9, Center: true},
	}
	const statusCol = 17

	sheets := []Sheet{
		{Name: "NFe Recebidas", Table: "RecebidasTable"},
		{Name: "NFe Emitidas", Table: "EmitidasTable"},
		{Name: "NFe Outras", Table: "OutrasTable"},
	}
	const (
		recebidas = iota
		emitidas
		outras
	)

	for _, row := range rows {
		cells := []any{
			string(row.CompanyRole),
			formatIssueDate(row.IssueDate),
			row.Competence,
			row.Serie,
			row.Numero,
			row.ChaveAcesso,
			cnpj.Format(row.EmitenteCNPJ),
			row.EmitenteName,
			row.EmitenteUF,
			cnpj.Format(row.DestinatarioCNPJ),
			row.DestinatarioName,
			row.Tipo,
			row.NatOp,
			row.Finalidade,
			row.TotalValue,
			row.ICMSValue,
			row.IPIValue,
			string(row.Situacao),
			string(row.Completeness),
			row.Manifestacao,
			row.Protocolo,
			row.EventCount,
		}

		i := outras
		switch row.CompanyRole {
		case nfe.CompanyRoleDestinatario:
			i = recebidas
		case nfe.CompanyRoleEmitente:
			i = emitidas
		}
		sheets[i].Rows = append(sheets[i].Rows, cells)
	}

	return Spreadsheet{Columns: columns, StatusCol: statusCol, Sheets: sheets}
}

// reais converts cents to reais.
func reais(m dfe.Money) float64 {
	return float64(m.Cents()) / 100.0
}

// tipoLabel names the tpNF code: 0 entrada, 1 saída.
func tipoLabel(tpNF string) string {
	switch tpNF {
	case "0":
		return "Entrada"
	case "1":
		return "Saída"
	default:
		return tpNF
	}
}

// finalidadeLabel names the finNFe code; an unknown code is kept as is.
func finalidadeLabel(finNFe string) string {
	switch finNFe {
	case "1":
		return "Normal"
	case "2":
		return "Complementar"
	case "3":
		return "Ajuste"
	case "4":
		return "Devolução"
	default:
		return finNFe
	}
}
