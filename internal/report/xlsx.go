package report

import (
	"fmt"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// NFSeXLSX lays out the NFS-e Excel export: one sheet per company role
// (emitidas, tomadas, intermediadas) plus one for documents without a known
// role, so every row lands somewhere.
func NFSeXLSX(rows []NFSeRow) Spreadsheet {
	columns := []Column{
		{Header: "Data Emissão", Width: 12, Center: true},
		{Header: "CNPJ/CPF Contraparte", Width: 20, Center: true},
		{Header: "Nome Contraparte", Width: 40, Center: true},
		{Header: "Nº NFSe", Width: 15, Center: true},
		{Header: "Competência", Width: 12, Center: true},
		{Header: "Chave de Acesso", Width: 45},
		{Header: "Valor Serviço (R$)", Width: 18, Money: true},
		{Header: "ISS (R$)", Width: 18, Money: true},
		{Header: "IRRF (R$)", Width: 18, Money: true},
		{Header: "INSS (R$)", Width: 18, Money: true},
		{Header: "PIS (R$)", Width: 18, Money: true},
		{Header: "COFINS (R$)", Width: 18, Money: true},
		{Header: "CSLL (R$)", Width: 18, Money: true},
		{Header: "Total Retenções (R$)", Width: 18, Money: true},
		{Header: "Valor Líquido Estimado (R$)", Width: 18, Money: true},
		{Header: "Situação", Width: 15, Center: true},
		{Header: "Descrição do Serviço", Width: 45},
		{Header: "Avisos do Parser", Width: 20},
	}

	sheets := []Sheet{
		{Name: "NFSe Emitidas", Table: "EmitidasTable"},
		{Name: "NFSe Tomadas", Table: "TomadasTable"},
		{Name: "NFSe Intermediadas", Table: "IntermediadasTable"},
		{Name: "NFSe Outras", Table: "OutrasTable"},
	}
	const (
		emitidas = iota
		tomadas
		intermediadas
		outras
	)

	for _, doc := range rows {
		warnings := ""
		if doc.WarningsCount > 0 {
			warnings = fmt.Sprintf("%d avisos", doc.WarningsCount)
		}
		cells := []any{
			formatIssueDate(doc.IssueDate),
			cnpj.Format(doc.CounterpartyCNPJ),
			doc.CounterpartyName,
			doc.NFSeNumber,
			doc.Competence,
			doc.ChaveAcesso,
			doc.ServiceValue,
			doc.ISSValue,
			doc.IRRFValue,
			doc.INSSValue,
			doc.PISValue,
			doc.COFINSValue,
			doc.CSLLValue,
			doc.TotalRetentions,
			doc.EstimatedNetValue,
			string(doc.Status),
			doc.Description,
			warnings,
		}

		i := outras
		switch doc.CompanyRole {
		case nfse.CompanyRolePrestada:
			i = emitidas
		case nfse.CompanyRoleTomada:
			i = tomadas
		case nfse.CompanyRoleIntermediario:
			i = intermediadas
		}
		sheets[i].Rows = append(sheets[i].Rows, cells)
	}

	return Spreadsheet{Columns: columns, StatusCol: 15, Sheets: sheets}
}
