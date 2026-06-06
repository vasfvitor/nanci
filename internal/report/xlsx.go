package report

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// GenerateXLSX creates a rich Excel spreadsheet with separated "Emitidas" and "Tomadas" sheets.
func GenerateXLSX(documents []nfse.CompanyDocument, outPath string) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	headers := []string{
		"Data Emissão", "CNPJ/CPF Contraparte", "Nome Contraparte", "Nº NFSe", "Competência", "Chave de Acesso",
		"Valor Serviço (R$)", "ISS (R$)", "IRRF (R$)", "INSS (R$)", "PIS (R$)", "COFINS (R$)", "CSLL (R$)",
		"Total Retenções (R$)", "Valor Líquido Estimado (R$)", "Situação", "Descrição do Serviço", "Avisos do Parser",
	}

	moneyStyle, err := f.NewStyle(&excelize.Style{NumFmt: 4})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "top", WrapText: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}
	centerStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}

	emitidas := []nfse.CompanyDocument{}
	tomadas := []nfse.CompanyDocument{}

	for _, doc := range documents {
		if doc.CompanyRole == "prestada" {
			emitidas = append(emitidas, doc)
		} else if doc.CompanyRole == "tomada" {
			tomadas = append(tomadas, doc)
		}
	}

	writeSheet := func(sheetName string, tableName string, docs []nfse.CompanyDocument, isEmitida bool) error {
		f.NewSheet(sheetName)

		// Set headers
		for col, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, 1)
			f.SetCellValue(sheetName, cell, header)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		// Populate rows
		for i, doc := range docs {
			row := i + 2

			issueStr := ""
			if !doc.IssueDate.IsZero() {
				issueStr = doc.IssueDate.Format(time.DateOnly)
			}

			docContraparte := doc.PrestadorCNPJ
			nomeContraparte := doc.PrestadorName
			if isEmitida {
				docContraparte = doc.TomadorCNPJ
				nomeContraparte = doc.TomadorName
			}

			valorLiquido := doc.ServiceValue - doc.TotalRetentions

			warnings := ""
			if len(doc.ParseWarnings) > 0 {
				warnings = fmt.Sprintf("%d avisos", len(doc.ParseWarnings))
			}

			rowValues := []interface{}{
				issueStr,
				cnpj.Format(docContraparte),
				nomeContraparte,
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
				valorLiquido,
				doc.Status,
				doc.ServiceDescription,
				warnings,
			}

			for col, val := range rowValues {
				cell, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cell, val)
			}

			// Apply styles
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), centerStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("P%d", row), fmt.Sprintf("P%d", row), centerStyle)
			f.SetCellStyle(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("O%d", row), moneyStyle)
		}

		// Column widths
		f.SetColWidth(sheetName, "A", "A", 12) // Data Emissao
		f.SetColWidth(sheetName, "B", "B", 20) // CNPJ
		f.SetColWidth(sheetName, "C", "C", 40) // Nome
		f.SetColWidth(sheetName, "D", "D", 15) // Numero
		f.SetColWidth(sheetName, "E", "E", 12) // Comp
		f.SetColWidth(sheetName, "F", "F", 45) // Chave
		f.SetColWidth(sheetName, "G", "O", 18) // Valores
		f.SetColWidth(sheetName, "P", "P", 15) // Status
		f.SetColWidth(sheetName, "Q", "Q", 45) // Descricao
		f.SetColWidth(sheetName, "R", "R", 20) // Avisos

		numRows := len(docs)
		if numRows > 0 {
			lastCell, _ := excelize.CoordinatesToCellName(len(headers), numRows+1)
			_ = f.AddTable(sheetName, &excelize.Table{
				Range:     fmt.Sprintf("A1:%s", lastCell),
				Name:      tableName,
				StyleName: "TableStyleLight15",
			})

			// Conditional formatting for CANCELADA
			formatID, _ := f.NewConditionalStyle(&excelize.Style{
				Font: &excelize.Font{Color: "#C00000"},
			})
			_ = f.SetConditionalFormat(sheetName, fmt.Sprintf("A2:%s", lastCell), []excelize.ConditionalFormatOptions{
				{
					Type:     "cell",
					Criteria: "==",
					Value:    `"cancelada"`,
					Format:   &formatID,
				},
				{
					Type:     "cell",
					Criteria: "==",
					Value:    `"substituida"`,
					Format:   &formatID,
				},
			})
		}
		return nil
	}

	if len(emitidas) > 0 {
		_ = writeSheet("NFSe Emitidas", "EmitidasTable", emitidas, true)
	}
	if len(tomadas) > 0 {
		_ = writeSheet("NFSe Tomadas", "TomadasTable", tomadas, false)
	}

	f.DeleteSheet("Sheet1")

	// If neither, just create an empty sheet
	if len(emitidas) == 0 && len(tomadas) == 0 {
		f.NewSheet("Documentos")
	}

	if err := f.SaveAs(outPath); err != nil {
		return fmt.Errorf("failed to save excel file: %w", err)
	}

	return nil
}
