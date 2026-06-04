package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/report"
	"github.com/vasfvitor/nanci/internal/store"
)

var (
	exportCNPJ       string
	exportCompetence string
	exportDirection  string
	exportOut        string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exporta os documentos sincronizados para planilhas ou ZIP",
}

var exportXlsxCmd = &cobra.Command{
	Use:   "xlsx",
	Short: "Exporta os dados para uma planilha Excel (.xlsx)",
	Run: func(cmd *cobra.Command, args []string) {
		docs := fetchExportDocs(cmd)
		
		fmt.Printf("Gerando arquivo Excel (%d documentos)...\n", len(docs))
		if err := report.GenerateXLSX(docs, exportOut); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao gerar arquivo XLSX: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Planilha gerada com sucesso: %s\n", exportOut)
	},
}

var exportCsvCmd = &cobra.Command{
	Use:   "csv",
	Short: "Exporta os dados para um arquivo de texto separado por vírgulas (.csv)",
	Run: func(cmd *cobra.Command, args []string) {
		docs := fetchExportDocs(cmd)
		
		fmt.Printf("Gerando arquivo CSV (%d documentos)...\n", len(docs))
		if err := report.GenerateCSV(docs, exportOut); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao gerar arquivo CSV: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Arquivo CSV gerado com sucesso: %s\n", exportOut)
	},
}

// fetchExportDocs is a helper function to avoid duplicating the query logic
func fetchExportDocs(cmd *cobra.Command) []nfse.Document {
	app, err := initApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := cnpj.Validate(exportCNPJ); err != nil {
		fmt.Fprintf(os.Stderr, "CNPJ inválido: %v\n", err)
		os.Exit(1)
	}

	cleanedCNPJ := cnpj.Clean(exportCNPJ)
	ctx := cmd.Context()

	company, err := app.Store.GetCompany(ctx, cleanedCNPJ)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar empresa: %v\n", err)
		os.Exit(1)
	}
	if company == nil {
		fmt.Fprintf(os.Stderr, "Empresa não encontrada para o CNPJ %s\n", cnpj.Format(cleanedCNPJ))
		os.Exit(1)
	}

	filter := store.DocumentFilter{
		Competence: exportCompetence,
		Direction:  exportDirection,
	}

	docs, err := app.Store.ListDocuments(ctx, company.ID, filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar documentos: %v\n", err)
		os.Exit(1)
	}

	if len(docs) == 0 {
		fmt.Println("Nenhum documento encontrado para exportar.")
		os.Exit(0)
	}

	return docs
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportXlsxCmd)
	exportCmd.AddCommand(exportCsvCmd)

	// Flags for xlsx
	exportXlsxCmd.Flags().StringVarP(&exportCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	exportXlsxCmd.Flags().StringVarP(&exportCompetence, "competencia", "m", "", "Competência (ex: 2026-06)")
	exportXlsxCmd.Flags().StringVarP(&exportDirection, "direcao", "d", "", "Direção (tomada, prestada, intermediario)")
	exportXlsxCmd.Flags().StringVarP(&exportOut, "out", "o", "export.xlsx", "Caminho do arquivo de saída")
	exportXlsxCmd.MarkFlagRequired("cnpj")

	// Flags for csv
	exportCsvCmd.Flags().StringVarP(&exportCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	exportCsvCmd.Flags().StringVarP(&exportCompetence, "competencia", "m", "", "Competência (ex: 2026-06)")
	exportCsvCmd.Flags().StringVarP(&exportDirection, "direcao", "d", "", "Direção (tomada, prestada, intermediario)")
	exportCsvCmd.Flags().StringVarP(&exportOut, "out", "o", "export.csv", "Caminho do arquivo de saída")
	exportCsvCmd.MarkFlagRequired("cnpj")
}
