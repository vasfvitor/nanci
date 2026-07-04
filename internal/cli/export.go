package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
)

func newExportCommand(env CommandEnv) *cobra.Command {
	var (
		cnpjFlag        string
		competenceFlag  string
		directionFlag   string
		incrementalFlag bool
	)
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta os documentos sincronizados para planilhas ou ZIP",
	}
	// Persistent flags inherited by every export subcommand.
	exportCmd.PersistentFlags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa")
	exportCmd.PersistentFlags().StringVarP(&competenceFlag, "competencia", "m", "", "Competência (ex: 2026-06)")
	exportCmd.PersistentFlags().StringVarP(&directionFlag, "direcao", "d", "", "Direção (tomada, prestada, intermediario)")
	exportCmd.PersistentFlags().BoolVar(&incrementalFlag, "incremental", false, "Exportar apenas documentos não exportados (ou modificados)")
	_ = exportCmd.MarkPersistentFlagRequired("cnpj")

	exportCmd.AddCommand(newExportXlsxCmd(env, &cnpjFlag, &competenceFlag, &directionFlag, &incrementalFlag))
	exportCmd.AddCommand(newExportCsvCmd(env, &cnpjFlag, &competenceFlag, &directionFlag, &incrementalFlag))
	exportCmd.AddCommand(newExportZipCmd(env, &cnpjFlag, &competenceFlag, &directionFlag, &incrementalFlag))
	exportCmd.AddCommand(newExportDANFSeCmd(env, &cnpjFlag))
	exportCmd.AddCommand(newExportDANFSeZipCmd(env, &cnpjFlag, &competenceFlag, &directionFlag, &incrementalFlag))
	return exportCmd
}

func newExportXlsxCmd(env CommandEnv, cnpj, competence, direction *string, incremental *bool) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "xlsx",
		Short: "Exporta os dados para uma planilha Excel (.xlsx)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			input := app.ExportInput{
				CNPJ:        *cnpj,
				Competence:  *competence,
				Direction:   *direction,
				OutPath:     outPath,
				Incremental: *incremental,
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando arquivo Excel...")
			res, err := application.Exports.ExportXLSX(cmd.Context(), input)
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo XLSX: %w", err)
			}

			printExportResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "export.xlsx", "Caminho do arquivo de saída")
	return cmd
}

func newExportCsvCmd(env CommandEnv, cnpj, competence, direction *string, incremental *bool) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "csv",
		Short: "Exporta os dados para um arquivo de texto separado por vírgulas (.csv)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			input := app.ExportInput{
				CNPJ:        *cnpj,
				Competence:  *competence,
				Direction:   *direction,
				OutPath:     outPath,
				Incremental: *incremental,
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando arquivo CSV...")
			res, err := application.Exports.ExportCSV(cmd.Context(), input)
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo CSV: %w", err)
			}

			printExportResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "export.csv", "Caminho do arquivo de saída")
	return cmd
}

func newExportZipCmd(env CommandEnv, cnpj, competence, direction *string, incremental *bool) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "zip",
		Short: "Exporta os arquivos físicos (.xml) em um arquivo compactado (.zip)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			input := app.ExportInput{
				CNPJ:        *cnpj,
				Competence:  *competence,
				Direction:   *direction,
				OutPath:     outPath,
				Incremental: *incremental,
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando arquivo ZIP...")
			res, err := application.Exports.ExportZIP(cmd.Context(), input)
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo ZIP: %w", err)
			}

			printExportResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "export.zip", "Caminho do arquivo de saída")
	return cmd
}

func newExportDANFSeCmd(env CommandEnv, cnpj *string) *cobra.Command {
	var (
		chave   string
		outPath string
	)
	cmd := &cobra.Command{
		Use:   "danfse",
		Short: "Exporta o DANFSe de uma NFS-e para PDF (.pdf)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			input := app.ExportDANFSeInput{
				CNPJ:        *cnpj,
				ChaveAcesso: chave,
				OutPath:     outPath,
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando DANFSe...")
			if err := application.Exports.ExportDANFSe(cmd.Context(), input); err != nil {
				return fmt.Errorf("erro ao gerar DANFSe: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "DANFSe gerado com sucesso: %s\n", outPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&chave, "chave", "", "Chave de acesso da NFS-e")
	cmd.Flags().StringVarP(&outPath, "out", "o", "danfse.pdf", "Caminho do arquivo de saída")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}

func newExportDANFSeZipCmd(env CommandEnv, cnpj, competence, direction *string, incremental *bool) *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "danfse-zip",
		Short: "Exporta DANFSes dos documentos filtrados em um arquivo compactado (.zip)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			input := app.ExportInput{
				CNPJ:        *cnpj,
				Competence:  *competence,
				Direction:   *direction,
				OutPath:     outPath,
				Incremental: *incremental,
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando ZIP de DANFSes...")
			res, err := application.Exports.ExportDANFSeZIP(cmd.Context(), input)
			if err != nil {
				return fmt.Errorf("erro ao gerar ZIP de DANFSes: %w", err)
			}

			printExportResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outPath, "out", "o", "danfses.zip", "Caminho do arquivo de saída")
	return cmd
}

func printExportResult(w io.Writer, res app.ExportResult) {
	if res.ExportedCount == 0 {
		if res.Incremental {
			_, _ = fmt.Fprintln(w, "Nenhum documento pendente para exportação incremental.")
		} else {
			_, _ = fmt.Fprintln(w, "Nenhum documento encontrado para exportar.")
		}
		return
	}
	_, _ = fmt.Fprintf(w, "Arquivo %s gerado com sucesso: %s\n", res.Format, res.OutPath)
	_, _ = fmt.Fprintf(w, "Documentos exportados: %d\n", res.ExportedCount)
}
