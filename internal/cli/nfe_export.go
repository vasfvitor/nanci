package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func newNFeExportCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta as NF-e sincronizadas (XML, CSV, Excel ou DANFE)",
	}
	exportCmd.AddCommand(newNFeBulkExportCmd(env, cnpjFlag, "zip",
		"Exporta o XML das NF-e e seus eventos em um arquivo compactado (.zip)", "nfe.zip", true,
		func(ctx context.Context, a *app.App, in app.NFeExportInput) (app.NFeExportResult, error) {
			return a.NFe.ExportXMLZip(ctx, in)
		}))
	exportCmd.AddCommand(newNFeBulkExportCmd(env, cnpjFlag, "csv",
		"Exporta as NF-e em uma planilha CSV", "nfe.csv", true,
		func(ctx context.Context, a *app.App, in app.NFeExportInput) (app.NFeExportResult, error) {
			return a.NFe.ExportCSV(ctx, in)
		}))
	exportCmd.AddCommand(newNFeBulkExportCmd(env, cnpjFlag, "xlsx",
		"Exporta as NF-e em uma planilha Excel (.xlsx)", "nfe.xlsx", true,
		func(ctx context.Context, a *app.App, in app.NFeExportInput) (app.NFeExportResult, error) {
			return a.NFe.ExportXLSX(ctx, in)
		}))
	exportCmd.AddCommand(newNFeBulkExportCmd(env, cnpjFlag, "danfe-zip",
		"Exporta o DANFE das NF-e completas em um arquivo compactado (.zip)", "danfes.zip", false,
		func(ctx context.Context, a *app.App, in app.NFeExportInput) (app.NFeExportResult, error) {
			return a.NFe.ExportDANFEZip(ctx, in)
		}))
	exportCmd.AddCommand(newNFeExportXMLCmd(env, cnpjFlag))
	exportCmd.AddCommand(newNFeExportDANFECmd(env, cnpjFlag))
	return exportCmd
}

// newNFeBulkExportCmd builds an export of the NF-e the filters select into
// one file at -o (default defaultOut). withResumos adds --incluir-resumos;
// without it the resumos are never exported and only counted.
func newNFeBulkExportCmd(
	env CommandEnv,
	cnpjFlag *string,
	use, short, defaultOut string,
	withResumos bool,
	run func(ctx context.Context, a *app.App, in app.NFeExportInput) (app.NFeExportResult, error),
) *cobra.Command {
	// The file type, "ZIP", "CSV" or "XLSX", names the file in the output.
	fileType := strings.ToUpper(strings.TrimPrefix(filepath.Ext(defaultOut), "."))
	var (
		competenceFlag     string
		papelFlag          string
		chaveFlags         []string
		incluirResumosFlag bool
		incrementalFlag    bool
		outPath            string
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Gerando arquivo %s...\n", fileType)
			res, err := run(cmd.Context(), application, app.NFeExportInput{
				CNPJ:           *cnpjFlag,
				Competence:     competenceFlag,
				Role:           papelFlag,
				ChavesAcesso:   chaveFlags,
				IncludeResumos: incluirResumosFlag,
				Incremental:    incrementalFlag,
				OutPath:        outPath,
			})
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo %s: %w", fileType, err)
			}

			printExportResult(cmd.OutOrStdout(), res.ExportResult)
			if res.SkippedResumos > 0 {
				if withResumos {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resumos não exportados: %d (use --incluir-resumos para incluí-los)\n", res.SkippedResumos)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resumos sem DANFE: %d\n", res.SkippedResumos)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Mês de emissão (ex: 2026-09)")
	cmd.Flags().StringVarP(&papelFlag, "papel", "p", "", "Papel da empresa (destinatario, emitente, transportador, autorizado, none)")
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Chave de acesso da NF-e (pode repetir)")
	if withResumos {
		cmd.Flags().BoolVar(&incluirResumosFlag, "incluir-resumos", false, "Inclui os resumos (resNFe), que não são o documento fiscal")
	}
	cmd.Flags().BoolVar(&incrementalFlag, "incremental", false, "Exportar apenas NF-e não exportadas (ou modificadas)")
	cmd.Flags().StringVarP(&outPath, "out", "o", defaultOut, "Caminho do arquivo de saída")
	return cmd
}

func newNFeExportXMLCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlag string
		outPath   string
	)
	cmd := &cobra.Command{
		Use:   "xml",
		Short: "Exporta o XML de uma NF-e (procNFe, ou resNFe de um resumo)",
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

			if err := application.NFe.ExportXML(cmd.Context(), app.NFeExportXMLInput{
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
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso da NF-e")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Caminho do arquivo de saída (padrão: <chave>.xml)")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}

func newNFeExportDANFECmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlag string
		outPath   string
	)
	cmd := &cobra.Command{
		Use:   "danfe",
		Short: "Exporta o DANFE (PDF) de uma NF-e completa",
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

			if err := application.NFe.ExportDANFE(cmd.Context(), app.NFeExportDANFEInput{
				CNPJ:        *cnpjFlag,
				ChaveAcesso: string(chave),
				OutPath:     outPath,
			}); err != nil {
				return fmt.Errorf("erro ao exportar DANFE: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "DANFE exportado com sucesso: %s\n", outPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso da NF-e")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Caminho do arquivo de saída (padrão: <chave>.pdf)")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}
