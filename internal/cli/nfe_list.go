package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newNFeListCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		competenceFlag   string
		situacaoFlag     string
		tipoFlag         string
		papelFlag        string
		manifestacaoFlag string
		emitenteFlag     string
		chaveFlags       []string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista as NF-e sincronizadas",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			docs, err := application.NFe.ListDocuments(cmd.Context(), app.NFeListInput{
				CNPJ:         *cnpjFlag,
				Competence:   competenceFlag,
				Situacao:     situacaoFlag,
				Completeness: tipoFlag,
				Role:         papelFlag,
				Manifestacao: manifestacaoFlag,
				EmitenteCNPJ: cnpj.Clean(emitenteFlag),
				ChavesAcesso: chaveFlags,
			})
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			if len(docs) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma NF-e encontrada.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "EMISSÃO\tCHAVE DE ACESSO\tPAPEL\tTIPO\tSITUAÇÃO\tEMITENTE\tNOME EMITENTE\tVALOR (R$)\tMANIFESTAÇÃO")
			_, _ = fmt.Fprintln(w, "-------\t---------------\t-----\t----\t--------\t--------\t-------------\t----------\t------------")
			for _, d := range docs {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					formatNFeDate(d.IssueDate),
					d.ChaveAcesso,
					d.CompanyRole,
					d.Completeness,
					d.Situacao,
					cnpj.Format(d.EmitenteCNPJ),
					truncateText(d.EmitenteName, 30),
					d.TotalValue.FormatBRL(),
					d.Manifestacao,
				)
			}
			_ = w.Flush()
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal de %d nota(s).\n", len(docs))
			return nil
		},
	}
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Filtrar pelo mês de emissão (ex: 2026-09)")
	cmd.Flags().StringVar(&situacaoFlag, "situacao", "", "Filtrar por situação (autorizada, denegada, cancelada)")
	cmd.Flags().StringVar(&tipoFlag, "tipo", "", "Filtrar por tipo (resumo, completa)")
	cmd.Flags().StringVar(&papelFlag, "papel", "", "Filtrar pelo papel da empresa (destinatario, emitente, transportador, autorizado, none)")
	cmd.Flags().StringVar(&manifestacaoFlag, "manifestacao", "", "Filtrar por manifestação (nenhuma, ciencia, confirmada, desconhecida, nao_realizada)")
	cmd.Flags().StringVar(&emitenteFlag, "emitente", "", "Filtrar pelo CNPJ ou CPF do emitente")
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Filtrar pela chave de acesso (pode repetir)")

	return cmd
}
