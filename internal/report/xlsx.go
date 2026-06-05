package report

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// GenerateXLSX creates an Excel spreadsheet from a list of company-facing documents and saves it to the specified path.
func GenerateXLSX(documents []nfse.CompanyDocument, outPath string) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheet := "Documentos"
	f.SetSheetName("Sheet1", sheet)

	// Set header
	headers := []string{
		"Competência", "Data Emissão", "Chave de Acesso", "Direção", "Visibilidade",
		"CNPJ Prestador", "Nome Prestador", "CNPJ Tomador", "Nome Tomador",
		"Valor Serviço (R$)", "ISS (R$)", "IRRF (R$)", "INSS (R$)",
		"PIS (R$)", "COFINS (R$)", "CSLL (R$)", "Status",
	}

	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Create style for money
	moneyStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 4, // 4 is typical format for #,##0.00
	})
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}

	// Populate rows
	for i, doc := range documents {
		row := i + 2 // 1-based index, row 1 is header

		issueStr := ""
		if !doc.IssueDate.IsZero() {
			issueStr = doc.IssueDate.Format(time.DateOnly)
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), doc.Competence)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), issueStr)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), doc.ChaveAcesso)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), doc.CompanyRole)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), doc.VisibilityReason)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), cnpj.Format(doc.PrestadorCNPJ))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), doc.PrestadorName)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), cnpj.Format(doc.TomadorCNPJ))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), doc.TomadorName)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), doc.ServiceValue)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), doc.ISSValue)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), doc.IRRFValue)
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), doc.INSSValue)
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), doc.PISValue)
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), doc.COFINSValue)
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), doc.CSLLValue)
		f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), doc.Status)

		// Apply money style to columns J to P
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("P%d", row), moneyStyle)
	}

	// Adjust column widths basic auto-fit approximation
	f.SetColWidth(sheet, "A", "A", 12)
	f.SetColWidth(sheet, "B", "B", 12)
	f.SetColWidth(sheet, "C", "C", 45)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 20)
	f.SetColWidth(sheet, "F", "F", 20)
	f.SetColWidth(sheet, "G", "G", 40)
	f.SetColWidth(sheet, "H", "H", 20)
	f.SetColWidth(sheet, "I", "I", 40)

	if err := f.SaveAs(outPath); err != nil {
		return fmt.Errorf("failed to save excel file: %w", err)
	}

	return nil
}
