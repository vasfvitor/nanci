package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func newCTeExportCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta os CT-e sincronizados em XML, planilha ou DACTE",
	}
	exportCmd.AddCommand(newCTeBulkExportCmd(env, cnpjFlag, "zip",
		"Exporta o XML dos CT-e e seus eventos em um arquivo compactado (.zip)", "ZIP", "cte.zip", (*app.CTeService).ExportXMLZip))
	exportCmd.AddCommand(newCTeBulkExportCmd(env, cnpjFlag, "csv",
		"Exporta os CT-e para uma planilha CSV (.csv)", "CSV", "cte.csv", (*app.CTeService).ExportCSV))
	exportCmd.AddCommand(newCTeBulkExportCmd(env, cnpjFlag, "xlsx",
		"Exporta os CT-e para uma planilha Excel (.xlsx)", "XLSX", "cte.xlsx", (*app.CTeService).ExportXLSX))
	exportCmd.AddCommand(newCTeBulkExportCmd(env, cnpjFlag, "dacte-zip",
		"Exporta o DACTE (PDF) dos CT-e modelo 57 em um arquivo compactado (.zip)", "ZIP", "dactes.zip", (*app.CTeService).ExportDACTEZip))
	exportCmd.AddCommand(newCTeExportXMLCmd(env, cnpjFlag))
	exportCmd.AddCommand(newCTeExportDACTECmd(env, cnpjFlag))
	return exportCmd
}

// newCTeBulkExportCmd builds an export of the CT-e that match the filters
// into one file. what names the file kind in the progress and error lines;
// run is the CTeService export to call.
func newCTeBulkExportCmd(env CommandEnv, cnpjFlag *string, use, short, what, defaultOut string,
	run func(*app.CTeService, context.Context, app.CTeExportInput) (app.CTeExportResult, error),
) *cobra.Command {
	var (
		competenceFlag  string
		papelFlag       string
		chaveFlags      []string
		incrementalFlag bool
		outPath         string
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Gerando arquivo %s...\n", what)
			res, err := run(application.CTe, cmd.Context(), app.CTeExportInput{
				CNPJ:         *cnpjFlag,
				Competence:   competenceFlag,
				Role:         papelFlag,
				ChavesAcesso: chaveFlags,
				Incremental:  incrementalFlag,
				OutPath:      outPath,
			})
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo %s: %w", what, err)
			}

			printExportResult(cmd.OutOrStdout(), res.ExportResult)
			if res.SkippedSemDACTE > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CT-e sem DACTE: %d\n", res.SkippedSemDACTE)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Mês de emissão (ex: 2026-09)")
	cmd.Flags().StringVarP(&papelFlag, "papel", "p", "", "Um papel da empresa, principal ou não (tomador, destinatario, remetente, expedidor, recebedor, emitente, autorizado, none)")
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Chave de acesso do CT-e (pode repetir)")
	cmd.Flags().BoolVar(&incrementalFlag, "incremental", false, "Exportar apenas CT-e não exportados (ou modificados)")
	cmd.Flags().StringVarP(&outPath, "out", "o", defaultOut, "Caminho do arquivo de saída")
	return cmd
}

func newCTeExportXMLCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlag string
		outPath   string
	)
	cmd := &cobra.Command{
		Use:   "xml",
		Short: "Exporta o XML de um CT-e (procCTe, procCTeOS, procGTVe ou procCTeSimp)",
		RunE: func(cmd *cobra.Command, args []string) error {
			chave, err := dfe.ParseAccessKey(chaveFlag)
			if err != nil {
				return fmt.Errorf("chave de acesso inválida: %w", err)
			}
			if outPath == "" {
				outPath = string(chave) + ".xml"
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			if err := application.CTe.ExportXML(cmd.Context(), app.CTeExportXMLInput{
				CNPJ:        *cnpjFlag,
				ChaveAcesso: string(chave),
				OutPath:     outPath,
			}); err != nil {
				return fmt.Errorf("erro ao exportar XML: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "XML exportado com sucesso: %s\n", outPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso do CT-e")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Caminho do arquivo de saída (padrão: <chave>.xml)")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}

func newCTeExportDACTECmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlag string
		outPath   string
	)
	cmd := &cobra.Command{
		Use:   "dacte",
		Short: "Exporta o DACTE (PDF) de um CT-e modelo 57",
		RunE: func(cmd *cobra.Command, args []string) error {
			chave, err := dfe.ParseAccessKey(chaveFlag)
			if err != nil {
				return fmt.Errorf("chave de acesso inválida: %w", err)
			}
			if outPath == "" {
				outPath = string(chave) + ".pdf"
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			if err := application.CTe.ExportDACTE(cmd.Context(), app.CTeExportDACTEInput{
				CNPJ:        *cnpjFlag,
				ChaveAcesso: string(chave),
				OutPath:     outPath,
			}); err != nil {
				return fmt.Errorf("erro ao exportar DACTE: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "DACTE exportado com sucesso: %s\n", outPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso do CT-e")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Caminho do arquivo de saída (padrão: <chave>.pdf)")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}
