package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/app"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		input := app.ExportInput{
			CNPJ:       exportCNPJ,
			Competence: exportCompetence,
			Direction:  exportDirection,
			OutPath:    exportOut,
		}

		fmt.Println("Gerando arquivo Excel...")
		if err := application.ExportXLSX(cmd.Context(), input); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao gerar arquivo XLSX: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Planilha gerada com sucesso: %s\n", exportOut)
		return nil
	},
}

var exportCsvCmd = &cobra.Command{
	Use:   "csv",
	Short: "Exporta os dados para um arquivo de texto separado por vírgulas (.csv)",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		input := app.ExportInput{
			CNPJ:       exportCNPJ,
			Competence: exportCompetence,
			Direction:  exportDirection,
			OutPath:    exportOut,
		}

		fmt.Println("Gerando arquivo CSV...")
		if err := application.ExportCSV(cmd.Context(), input); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao gerar arquivo CSV: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Arquivo CSV gerado com sucesso: %s\n", exportOut)
		return nil
	},
}

var exportZipCmd = &cobra.Command{
	Use:   "zip",
	Short: "Exporta os arquivos físicos (.xml) em um arquivo compactado (.zip)",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		input := app.ExportInput{
			CNPJ:       exportCNPJ,
			Competence: exportCompetence,
			Direction:  exportDirection,
			OutPath:    exportOut,
		}

		fmt.Println("Gerando arquivo ZIP...")
		if err := application.ExportZIP(cmd.Context(), input); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao gerar arquivo ZIP: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Arquivo ZIP gerado com sucesso: %s\n", exportOut)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportXlsxCmd)
	exportCmd.AddCommand(exportCsvCmd)
	exportCmd.AddCommand(exportZipCmd)

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

	// Flags for zip
	exportZipCmd.Flags().StringVarP(&exportCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	exportZipCmd.Flags().StringVarP(&exportCompetence, "competencia", "m", "", "Competência (ex: 2026-06)")
	exportZipCmd.Flags().StringVarP(&exportDirection, "direcao", "d", "", "Direção (tomada, prestada, intermediario)")
	exportZipCmd.Flags().StringVarP(&exportOut, "out", "o", "export.zip", "Caminho do arquivo de saída")
	exportZipCmd.MarkFlagRequired("cnpj")
}
