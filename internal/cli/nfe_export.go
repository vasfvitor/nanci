package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func newNFeExportCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta o XML das NF-e sincronizadas",
	}
	exportCmd.AddCommand(newNFeExportZipCmd(env, cnpjFlag))
	exportCmd.AddCommand(newNFeExportXMLCmd(env, cnpjFlag))
	return exportCmd
}

func newNFeExportZipCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		competenceFlag     string
		papelFlag          string
		chaveFlags         []string
		incluirResumosFlag bool
		incrementalFlag    bool
		outPath            string
	)
	cmd := &cobra.Command{
		Use:   "zip",
		Short: "Exporta o XML das NF-e e seus eventos em um arquivo compactado (.zip)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando arquivo ZIP...")
			res, err := application.NFe.ExportXMLZip(cmd.Context(), app.NFeExportInput{
				CNPJ:           *cnpjFlag,
				Competence:     competenceFlag,
				Role:           papelFlag,
				ChavesAcesso:   chaveFlags,
				IncludeResumos: incluirResumosFlag,
				Incremental:    incrementalFlag,
				OutPath:        outPath,
			})
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo ZIP: %w", err)
			}

			printExportResult(cmd.OutOrStdout(), res.ExportResult)
			if res.SkippedResumos > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resumos não exportados: %d (use --incluir-resumos para incluí-los)\n", res.SkippedResumos)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Mês de emissão (ex: 2026-09)")
	cmd.Flags().StringVarP(&papelFlag, "papel", "p", "", "Papel da empresa (destinatario, emitente, transportador, autorizado, none)")
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Chave de acesso da NF-e (pode repetir)")
	cmd.Flags().BoolVar(&incluirResumosFlag, "incluir-resumos", false, "Inclui os resumos (resNFe), que não são o documento fiscal")
	cmd.Flags().BoolVar(&incrementalFlag, "incremental", false, "Exportar apenas NF-e não exportadas (ou modificadas)")
	cmd.Flags().StringVarP(&outPath, "out", "o", "nfe.zip", "Caminho do arquivo de saída")
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
