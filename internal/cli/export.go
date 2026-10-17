package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func newExportCommand(env CommandEnv) *cobra.Command {
	flags := &nfseExportFlags{}
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta os documentos sincronizados para planilhas ou ZIP",
	}
	// Persistent flags inherited by every export subcommand.
	exportCmd.PersistentFlags().StringVarP(&flags.cnpj, "cnpj", "c", "", "CNPJ da empresa")
	exportCmd.PersistentFlags().StringVarP(&flags.competence, "competencia", "m", "", "Competência (ex: 2026-06)")
	exportCmd.PersistentFlags().StringVarP(&flags.direction, "direcao", "d", "", "Direção (tomada, prestada, intermediario)")
	exportCmd.PersistentFlags().BoolVar(&flags.incremental, "incremental", false, "Exportar apenas documentos não exportados (ou modificados)")
	_ = exportCmd.MarkPersistentFlagRequired("cnpj")

	exportCmd.AddCommand(newNFSeBulkExportCmd(env, flags, "xlsx",
		"Exporta os dados para uma planilha Excel (.xlsx)", "export.xlsx", "Gerando arquivo Excel...", "erro ao gerar arquivo XLSX",
		func(ctx context.Context, a *app.App, in app.ExportInput) (app.ExportResult, error) {
			return a.Exports.ExportXLSX(ctx, in)
		}))
	exportCmd.AddCommand(newNFSeBulkExportCmd(env, flags, "csv",
		"Exporta os dados para um arquivo de texto separado por vírgulas (.csv)", "export.csv", "Gerando arquivo CSV...", "erro ao gerar arquivo CSV",
		func(ctx context.Context, a *app.App, in app.ExportInput) (app.ExportResult, error) {
			return a.Exports.ExportCSV(ctx, in)
		}))
	exportCmd.AddCommand(newNFSeBulkExportCmd(env, flags, "zip",
		"Exporta os arquivos físicos (.xml) em um arquivo compactado (.zip)", "export.zip", "Gerando arquivo ZIP...", "erro ao gerar arquivo ZIP",
		func(ctx context.Context, a *app.App, in app.ExportInput) (app.ExportResult, error) {
			return a.Exports.ExportZIP(ctx, in)
		}))
	exportCmd.AddCommand(newExportXMLCmd(env, &flags.cnpj))
	exportCmd.AddCommand(newExportDANFSeCmd(env, &flags.cnpj))
	exportCmd.AddCommand(newNFSeBulkExportCmd(env, flags, "danfse-zip",
		"Exporta DANFSes dos documentos filtrados em um arquivo compactado (.zip)", "danfses.zip", "Gerando ZIP de DANFSes...", "erro ao gerar ZIP de DANFSes",
		func(ctx context.Context, a *app.App, in app.ExportInput) (app.ExportResult, error) {
			return a.Exports.ExportDANFSeZIP(ctx, in)
		}))
	return exportCmd
}

// nfseExportFlags holds the persistent flags of the export group.
type nfseExportFlags struct {
	cnpj        string
	competence  string
	direction   string
	incremental bool
}

// newNFSeBulkExportCmd builds an export of every document matching the group
// filters and the repeatable --chave. progress is printed before run calls
// the use case; failure prefixes its error.
func newNFSeBulkExportCmd(env CommandEnv, flags *nfseExportFlags, use, short, defaultOut, progress, failure string, run func(context.Context, *app.App, app.ExportInput) (app.ExportResult, error)) *cobra.Command {
	var (
		chaveFlags []string
		outPath    string
	)
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, raw := range chaveFlags {
				if _, err := parseNFSeChave(raw); err != nil {
					return err
				}
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), progress)
			res, err := run(cmd.Context(), application, app.ExportInput{
				CNPJ:         flags.cnpj,
				Competence:   flags.competence,
				Direction:    flags.direction,
				OutPath:      outPath,
				Incremental:  flags.incremental,
				ChavesAcesso: chaveFlags,
			})
			if err != nil {
				return fmt.Errorf("%s: %w", failure, err)
			}

			printExportResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Chave de acesso da NFS-e (pode repetir)")
	cmd.Flags().StringVarP(&outPath, "out", "o", defaultOut, "Caminho do arquivo de saída")
	return cmd
}

func newExportXMLCmd(env CommandEnv, cnpj *string) *cobra.Command {
	var (
		chaveFlag string
		outPath   string
	)
	cmd := &cobra.Command{
		Use:   "xml",
		Short: "Exporta o XML de uma NFS-e",
		RunE: func(cmd *cobra.Command, args []string) error {
			chave, err := parseNFSeChave(chaveFlag)
			if err != nil {
				return err
			}
			if outPath == "" {
				outPath = chave + ".xml"
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			if err := application.Exports.ExportXML(cmd.Context(), app.ExportXMLInput{
				CNPJ:        *cnpj,
				ChaveAcesso: chave,
				OutPath:     outPath,
			}); err != nil {
				return fmt.Errorf("erro ao exportar XML: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "XML exportado com sucesso: %s\n", outPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso da NFS-e")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Caminho do arquivo de saída (padrão: <chave>.xml)")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}

// parseNFSeChave checks a --chave value before the database is opened and
// returns it trimmed. The message matches the one app gives.
func parseNFSeChave(raw string) (string, error) {
	chave := strings.TrimSpace(raw)
	if _, err := nfse.ParseInfNFSeID(chave); err != nil {
		return "", fmt.Errorf("chave de acesso inválida %q: %w", chave, err)
	}
	return chave, nil
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
