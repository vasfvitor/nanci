package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func newCTeExportCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Exporta o XML dos CT-e sincronizados",
	}
	exportCmd.AddCommand(newCTeExportZipCmd(env, cnpjFlag))
	exportCmd.AddCommand(newCTeExportXMLCmd(env, cnpjFlag))
	return exportCmd
}

func newCTeExportZipCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		competenceFlag  string
		papelFlag       string
		chaveFlags      []string
		incrementalFlag bool
		outPath         string
	)
	cmd := &cobra.Command{
		Use:   "zip",
		Short: "Exporta o XML dos CT-e e seus eventos em um arquivo compactado (.zip)",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Gerando arquivo ZIP...")
			res, err := application.CTe.ExportXMLZip(cmd.Context(), app.CTeExportInput{
				CNPJ:         *cnpjFlag,
				Competence:   competenceFlag,
				Role:         papelFlag,
				ChavesAcesso: chaveFlags,
				Incremental:  incrementalFlag,
				OutPath:      outPath,
			})
			if err != nil {
				return fmt.Errorf("erro ao gerar arquivo ZIP: %w", err)
			}

			printExportResult(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Mês de emissão (ex: 2026-09)")
	cmd.Flags().StringVarP(&papelFlag, "papel", "p", "", "Um papel da empresa, principal ou não (tomador, destinatario, remetente, expedidor, recebedor, emitente, autorizado, none)")
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Chave de acesso do CT-e (pode repetir)")
	cmd.Flags().BoolVar(&incrementalFlag, "incremental", false, "Exportar apenas CT-e não exportados (ou modificados)")
	cmd.Flags().StringVarP(&outPath, "out", "o", "cte.zip", "Caminho do arquivo de saída")
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
