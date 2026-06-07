package report

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// GenerateXLSX creates a rich Excel spreadsheet with separated "Emitidas" and "Tomadas" sheets.
func GenerateXLSX(rows []ReportRow, outPath string) error {
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

	emitidas := []ReportRow{}
	tomadas := []ReportRow{}

	for _, r := range rows {
		switch r.CompanyRole {
		case nfse.CompanyRolePrestada:
			emitidas = append(emitidas, r)
		case nfse.CompanyRoleTomada:
			tomadas = append(tomadas, r)
		}
	}

	writeSheet := func(sheetName string, tableName string, docs []ReportRow, isEmitida bool) error {
		_, _ = f.NewSheet(sheetName)

		// Set headers
		for col, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, 1)
			_ = f.SetCellValue(sheetName, cell, header)
			_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		// Populate rows
		for i, doc := range docs {
			row := i + 2

			issueStr := ""
			if !doc.IssueDate.IsZero() {
				issueStr = doc.IssueDate.Format(time.DateOnly)
			}

			warnings := ""
			if doc.WarningsCount > 0 {
				warnings = fmt.Sprintf("%d avisos", doc.WarningsCount)
			}

			rowValues := []interface{}{
				issueStr,
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

			for col, val := range rowValues {
				cell, _ := excelize.CoordinatesToCellName(col+1, row)
				_ = f.SetCellValue(sheetName, cell, val)
			}

			// Apply styles
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("E%d", row), centerStyle)
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("P%d", row), fmt.Sprintf("P%d", row), centerStyle)
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("O%d", row), moneyStyle)
		}

		setColWidth := func(start, end string, width float64) error {
			if err := f.SetColWidth(sheetName, start, end, width); err != nil {
				return fmt.Errorf("failed to set width %s:%s on %s: %w", start, end, sheetName, err)
			}
			return nil
		}

		if err := setColWidth("A", "A", 12); err != nil {
			return err
		} // Data Emissao
		if err := setColWidth("B", "B", 20); err != nil {
			return err
		} // CNPJ
		if err := setColWidth("C", "C", 40); err != nil {
			return err
		} // Nome
		if err := setColWidth("D", "D", 15); err != nil {
			return err
		} // Numero
		if err := setColWidth("E", "E", 12); err != nil {
			return err
		} // Comp
		if err := setColWidth("F", "F", 45); err != nil {
			return err
		} // Chave
		if err := setColWidth("G", "O", 18); err != nil {
			return err
		} // Valores
		if err := setColWidth("P", "P", 15); err != nil {
			return err
		} // Status
		if err := setColWidth("Q", "Q", 45); err != nil {
			return err
		} // Descricao
		if err := setColWidth("R", "R", 20); err != nil {
			return err
		} // Avisos

		numRows := len(docs)
		if numRows > 0 {
			lastCell, _ := excelize.CoordinatesToCellName(len(headers), numRows+1)
			if err := f.AddTable(sheetName, &excelize.Table{
				Range:     fmt.Sprintf("A1:%s", lastCell),
				Name:      tableName,
				StyleName: "TableStyleLight15",
			}); err != nil {
				return fmt.Errorf("failed to add table: %w", err)
			}

			cancelledFormatID, err := f.NewConditionalStyle(&excelize.Style{
				Font: &excelize.Font{Color: "#C00000"},
			})
			if err != nil {
				return fmt.Errorf("failed to create cancelled conditional style: %w", err)
			}

			substitutedFormatID, err := f.NewConditionalStyle(&excelize.Style{
				Font: &excelize.Font{Color: "#666666", Italic: true},
			})
			if err != nil {
				return fmt.Errorf("failed to create substituted conditional style: %w", err)
			}

			dataRange := fmt.Sprintf("A2:%s", lastCell)
			err = f.SetConditionalFormat(sheetName, dataRange, []excelize.ConditionalFormatOptions{
				{
					Type:     "formula",
					Criteria: `=$P2="cancelada"`,
					Format:   &cancelledFormatID,
				},
				{
					Type:     "formula",
					Criteria: `=$P2="substituida"`,
					Format:   &substitutedFormatID,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to set conditional format on %s: %w", sheetName, err)
			}
		}
		return nil
	}

	if len(emitidas) > 0 {
		if err := writeSheet("NFSe Emitidas", "EmitidasTable", emitidas, true); err != nil {
			return err
		}
	}
	if len(tomadas) > 0 {
		if err := writeSheet("NFSe Tomadas", "TomadasTable", tomadas, false); err != nil {
			return err
		}
	}

	_ = f.DeleteSheet("Sheet1")

	// If neither, just create an empty sheet
	if len(emitidas) == 0 && len(tomadas) == 0 {
		_, _ = f.NewSheet("Documentos")
	}

	if err := f.SaveAs(outPath); err != nil {
		return fmt.Errorf("failed to save excel file: %w", err)
	}

	return nil
}
