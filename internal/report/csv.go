package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// GenerateCSV creates a CSV file from a list of company-facing documents and saves it to the specified path.
func GenerateCSV(documents []ReportRow, outPath string) (err error) {
	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create csv file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close csv file: %w", cerr)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"Competencia", "Data Emissao", "Chave de Acesso", "Direcao", "Visibilidade",
		"CNPJ Prestador", "Nome Prestador", "CNPJ Tomador", "Nome Tomador",
		"Valor Servico", "ISS", "IRRF", "INSS", "PIS", "COFINS", "CSLL", "Status",
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	// Write rows
	for _, doc := range documents {
		issueStr := ""
		if !doc.IssueDate.IsZero() {
			issueStr = doc.IssueDate.Format(time.DateOnly)
		}

		row := []string{
			doc.Competence,
			issueStr,
			doc.ChaveAcesso,
			string(doc.CompanyRole),
			"", // Visibility is no longer easily available in ReportRow since it was removed for simplicity, leaving blank
			cnpj.Format(doc.PrestadorCNPJ),
			doc.PrestadorName,
			cnpj.Format(doc.TomadorCNPJ),
			doc.TomadorName,
			fmt.Sprintf("%.2f", doc.ServiceValue),
			fmt.Sprintf("%.2f", doc.ISSValue),
			fmt.Sprintf("%.2f", doc.IRRFValue),
			fmt.Sprintf("%.2f", doc.INSSValue),
			fmt.Sprintf("%.2f", doc.PISValue),
			fmt.Sprintf("%.2f", doc.COFINSValue),
			fmt.Sprintf("%.2f", doc.CSLLValue),
			string(doc.Status),
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	return nil
}
