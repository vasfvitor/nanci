package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
)

var (
	exportCNPJ        string
	exportCompetence  string
	exportDirection   string
	exportOut         string
	exportChave       string
	exportIncremental bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exporta os documentos sincronizados para planilhas ou ZIP",
}

func printExportResult(res app.ExportResult) {
	if res.ExportedCount == 0 {
		if res.Incremental {
			fmt.Println("Nenhum documento pendente para exportação incremental.")
		} else {
			fmt.Println("Nenhum documento encontrado para exportar.")
		}
		return
	}
	fmt.Printf("Arquivo %s gerado com sucesso: %s\n", res.Format, res.OutPath)
	fmt.Printf("Documentos exportados: %d\n", res.ExportedCount)
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
			CNPJ:        exportCNPJ,
			Competence:  exportCompetence,
			Direction:   exportDirection,
			OutPath:     exportOut,
			Incremental: exportIncremental,
		}

		fmt.Println("Gerando arquivo Excel...")
		res, err := application.ExportXLSX(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("erro ao gerar arquivo XLSX: %w", err)
		}

		printExportResult(res)
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
			CNPJ:        exportCNPJ,
			Competence:  exportCompetence,
			Direction:   exportDirection,
			OutPath:     exportOut,
			Incremental: exportIncremental,
		}

		fmt.Println("Gerando arquivo CSV...")
		res, err := application.ExportCSV(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("erro ao gerar arquivo CSV: %w", err)
		}

		printExportResult(res)
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
			CNPJ:        exportCNPJ,
			Competence:  exportCompetence,
			Direction:   exportDirection,
			OutPath:     exportOut,
			Incremental: exportIncremental,
		}

		fmt.Println("Gerando arquivo ZIP...")
		res, err := application.ExportZIP(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("erro ao gerar arquivo ZIP: %w", err)
		}

		printExportResult(res)
		return nil
	},
}

var exportDANFSeCmd = &cobra.Command{
	Use:   "danfse",
	Short: "Exporta o DANFSe de uma NFS-e para PDF (.pdf)",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		input := app.ExportDANFSeInput{
			CNPJ:        exportCNPJ,
			ChaveAcesso: exportChave,
			OutPath:     exportOut,
		}

		fmt.Println("Gerando DANFSe...")
		if err := application.ExportDANFSe(cmd.Context(), input); err != nil {
			return fmt.Errorf("erro ao gerar DANFSe: %w", err)
		}

		fmt.Printf("DANFSe gerado com sucesso: %s\n", exportOut)
		return nil
	},
}

var exportDANFSeZipCmd = &cobra.Command{
	Use:   "danfse-zip",
	Short: "Exporta DANFSes dos documentos filtrados em um arquivo compactado (.zip)",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		input := app.ExportInput{
			CNPJ:        exportCNPJ,
			Competence:  exportCompetence,
			Direction:   exportDirection,
			OutPath:     exportOut,
			Incremental: exportIncremental,
		}

		fmt.Println("Gerando ZIP de DANFSes...")
		res, err := application.ExportDANFSeZIP(cmd.Context(), input)
		if err != nil {
			return fmt.Errorf("erro ao gerar ZIP de DANFSes: %w", err)
		}

		printExportResult(res)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportXlsxCmd)
	exportCmd.AddCommand(exportCsvCmd)
	exportCmd.AddCommand(exportZipCmd)
	exportCmd.AddCommand(exportDANFSeCmd)
	exportCmd.AddCommand(exportDANFSeZipCmd)

	// Persistent flags applied to all subcommands
	exportCmd.PersistentFlags().StringVarP(&exportCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	exportCmd.PersistentFlags().StringVarP(&exportCompetence, "competencia", "m", "", "Competência (ex: 2026-06)")
	exportCmd.PersistentFlags().StringVarP(&exportDirection, "direcao", "d", "", "Direção (tomada, prestada, intermediario)")
	exportCmd.PersistentFlags().BoolVar(&exportIncremental, "incremental", false, "Exportar apenas documentos não exportados (ou modificados)")
	_ = exportCmd.MarkPersistentFlagRequired("cnpj")

	// Specific flags for out
	exportXlsxCmd.Flags().StringVarP(&exportOut, "out", "o", "export.xlsx", "Caminho do arquivo de saída")
	exportCsvCmd.Flags().StringVarP(&exportOut, "out", "o", "export.csv", "Caminho do arquivo de saída")
	exportZipCmd.Flags().StringVarP(&exportOut, "out", "o", "export.zip", "Caminho do arquivo de saída")
	exportDANFSeCmd.Flags().StringVar(&exportChave, "chave", "", "Chave de acesso da NFS-e")
	exportDANFSeCmd.Flags().StringVarP(&exportOut, "out", "o", "danfse.pdf", "Caminho do arquivo de saída")
	_ = exportDANFSeCmd.MarkFlagRequired("chave")
	exportDANFSeZipCmd.Flags().StringVarP(&exportOut, "out", "o", "danfses.zip", "Caminho do arquivo de saída")
}
