package report

import (
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// CTeRow is one CT-e in the spreadsheet exports, with values in reais.
type CTeRow struct {
	CompanyRole         cte.CompanyRole
	Papeis              []cte.CompanyRole
	IssueDate           time.Time
	Competence          string
	Modelo              string
	Serie               string
	Numero              string
	ChaveAcesso         string
	CFOP                string
	NatOp               string
	Inicio              string // município/UF
	Fim                 string // município/UF
	EmitenteCNPJ        string
	EmitenteName        string
	TomadorCNPJ         string
	TomadorName         string
	RemetenteCNPJ       string
	RemetenteName       string
	DestinatarioCNPJ    string
	DestinatarioName    string
	TotalValue          float64
	ReceivableValue     float64
	ICMSValue           float64
	TotTribValue        float64
	CargaValue          float64
	ProdutoPredominante string
	NFeCount            int
	Situacao            cte.Situacao
	Protocolo           string
	EventCount          int
}

// BuildCTeRows turns company CT-e into spreadsheet rows, in order.
func BuildCTeRows(docs []cte.CompanyDocument) []CTeRow {
	rows := make([]CTeRow, 0, len(docs))
	for _, doc := range docs {
		rows = append(rows, CTeRow{
			CompanyRole:         doc.CompanyRole,
			Papeis:              doc.Papeis,
			IssueDate:           doc.IssueDate,
			Competence:          doc.Competence,
			Modelo:              doc.Modelo,
			Serie:               doc.Serie,
			Numero:              doc.Numero,
			ChaveAcesso:         string(doc.ChaveAcesso),
			CFOP:                doc.CFOP,
			NatOp:               doc.NatOp,
			Inicio:              municipioUF(doc.MunIni),
			Fim:                 municipioUF(doc.MunFim),
			EmitenteCNPJ:        doc.Emitente.CNPJ,
			EmitenteName:        doc.Emitente.Name,
			TomadorCNPJ:         doc.Tomador.CNPJ,
			TomadorName:         doc.Tomador.Name,
			RemetenteCNPJ:       doc.Remetente.CNPJ,
			RemetenteName:       doc.Remetente.Name,
			DestinatarioCNPJ:    doc.Destinatario.CNPJ,
			DestinatarioName:    doc.Destinatario.Name,
			TotalValue:          reais(doc.TotalValue),
			ReceivableValue:     reais(doc.ReceivableValue),
			ICMSValue:           reais(doc.ICMSValue),
			TotTribValue:        reais(doc.TotTribValue),
			CargaValue:          reais(doc.CargaValue),
			ProdutoPredominante: doc.ProdutoPredominante,
			NFeCount:            len(doc.NFeChaves),
			Situacao:            doc.Situacao,
			Protocolo:           doc.Protocolo,
			EventCount:          doc.EventCount,
		})
	}
	return rows
}

// CTeSpreadsheet lays out the CT-e CSV and Excel exports: one sheet for the
// CT-e the company took (tomador), one for those it issued (emitente) and one
// for every other role, so every row lands somewhere. The CSV writes the
// three sheets in that order under one header.
func CTeSpreadsheet(rows []CTeRow) Spreadsheet {
	columns := []Column{
		{Header: "Papel", Width: 14, Center: true},
		{Header: "Papéis", Width: 24},
		{Header: "Data Emissão", Width: 12, Center: true},
		{Header: "Competência", Width: 12, Center: true},
		{Header: "Modelo", Width: 8, Center: true},
		{Header: "Série", Width: 8, Center: true},
		{Header: "Número", Width: 12, Center: true},
		{Header: "Chave de Acesso", Width: 47},
		{Header: "CFOP", Width: 8, Center: true},
		{Header: "Natureza da Operação", Width: 35},
		{Header: "Início (município/UF)", Width: 25},
		{Header: "Fim (município/UF)", Width: 25},
		{Header: "CNPJ/CPF Emitente", Width: 20, Center: true},
		{Header: "Nome Emitente", Width: 40},
		{Header: "CNPJ/CPF Tomador", Width: 20, Center: true},
		{Header: "Nome Tomador", Width: 40},
		{Header: "CNPJ/CPF Remetente", Width: 20, Center: true},
		{Header: "Nome Remetente", Width: 40},
		{Header: "CNPJ/CPF Destinatário", Width: 20, Center: true},
		{Header: "Nome Destinatário", Width: 40},
		{Header: "Valor Prestação (R$)", Width: 18, Money: true},
		{Header: "Valor a Receber (R$)", Width: 18, Money: true},
		{Header: "ICMS (R$)", Width: 18, Money: true},
		{Header: "Tributos Aprox. (R$)", Width: 18, Money: true},
		{Header: "Valor Carga (R$)", Width: 18, Money: true},
		{Header: "Produto Predominante", Width: 30},
		{Header: "NF-e Transportadas", Width: 12, Center: true},
		{Header: "Situação", Width: 15, Center: true},
		{Header: "Protocolo", Width: 18, Center: true},
		{Header: "Eventos", Width: 10, Center: true},
	}
	const statusCol = 27

	sheets := []Sheet{
		{Name: "CTe Tomados", Table: "CTeTomadosTable"},
		{Name: "CTe Emitidos", Table: "CTeEmitidosTable"},
		{Name: "CTe Outros", Table: "CTeOutrosTable"},
	}
	const (
		tomados = iota
		emitidos
		outros
	)

	for _, row := range rows {
		papeis := make([]string, len(row.Papeis))
		for i, p := range row.Papeis {
			papeis[i] = string(p)
		}
		cells := []any{
			string(row.CompanyRole),
			strings.Join(papeis, ", "),
			formatIssueDate(row.IssueDate),
			row.Competence,
			row.Modelo,
			row.Serie,
			row.Numero,
			row.ChaveAcesso,
			row.CFOP,
			row.NatOp,
			row.Inicio,
			row.Fim,
			cnpj.Format(row.EmitenteCNPJ),
			row.EmitenteName,
			cnpj.Format(row.TomadorCNPJ),
			row.TomadorName,
			cnpj.Format(row.RemetenteCNPJ),
			row.RemetenteName,
			cnpj.Format(row.DestinatarioCNPJ),
			row.DestinatarioName,
			row.TotalValue,
			row.ReceivableValue,
			row.ICMSValue,
			row.TotTribValue,
			row.CargaValue,
			row.ProdutoPredominante,
			row.NFeCount,
			string(row.Situacao),
			row.Protocolo,
			row.EventCount,
		}

		i := outros
		switch row.CompanyRole {
		case cte.CompanyRoleTomador:
			i = tomados
		case cte.CompanyRoleEmitente:
			i = emitidos
		}
		sheets[i].Rows = append(sheets[i].Rows, cells)
	}

	return Spreadsheet{Columns: columns, StatusCol: statusCol, Sheets: sheets}
}

// municipioUF is "município/UF", either part alone when the other is
// missing, or empty.
func municipioUF(m cte.Municipio) string {
	switch {
	case m.Nome != "" && m.UF != "":
		return m.Nome + "/" + m.UF
	case m.Nome != "":
		return m.Nome
	default:
		return m.UF
	}
}
