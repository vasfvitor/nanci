package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newCTeListCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		competenceFlag string
		situacaoFlag   string
		papelFlag      string
		modeloFlag     string
		emitenteFlag   string
		tomadorFlag    string
		nfeFlag        string
		chaveFlags     []string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista os CT-e sincronizados",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			docs, err := application.CTe.ListDocuments(cmd.Context(), app.ListCTeInput{
				CNPJ:         *cnpjFlag,
				Competence:   competenceFlag,
				Situacao:     situacaoFlag,
				Role:         papelFlag,
				Modelo:       modeloFlag,
				EmitenteCNPJ: cnpj.Clean(emitenteFlag),
				TomadorCNPJ:  cnpj.Clean(tomadorFlag),
				NFeChave:     nfeFlag,
				ChavesAcesso: chaveFlags,
			})
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			if len(docs) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Nenhum CT-e encontrado.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "CHAVE DE ACESSO\tMODELO\tNÚMERO/SÉRIE\tEMISSÃO\tEMITENTE\tNOME EMITENTE\tTOMADOR\tPAPEL\tVALOR (R$)\tSITUAÇÃO")
			_, _ = fmt.Fprintln(w, "---------------\t------\t------------\t-------\t--------\t-------------\t-------\t-----\t----------\t--------")
			for _, d := range docs {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s/%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					d.ChaveAcesso,
					d.Modelo,
					d.Numero,
					d.Serie,
					formatNFeDate(d.IssueDate),
					cnpj.Format(d.Emitente.CNPJ),
					truncateText(d.Emitente.Name, 30),
					dashIfEmpty(cnpj.Format(d.Tomador.CNPJ)),
					ctePapelLabel(d),
					d.TotalValue.FormatBRL(),
					d.Situacao,
				)
			}
			_ = w.Flush()
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal de %d CT-e listado(s).\n", len(docs))
			return nil
		},
	}
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Filtrar pelo mês de emissão (ex: 2026-09)")
	cmd.Flags().StringVar(&situacaoFlag, "situacao", "", "Filtrar por situação (autorizada, denegada, cancelada)")
	cmd.Flags().StringVarP(&papelFlag, "papel", "p", "", "Filtrar por um papel da empresa, principal ou não (tomador, destinatario, remetente, expedidor, recebedor, emitente, autorizado, none)")
	cmd.Flags().StringVar(&modeloFlag, "modelo", "", "Filtrar pelo modelo (57 CT-e, 64 GTV-e, 67 CT-e OS)")
	cmd.Flags().StringVar(&emitenteFlag, "emitente", "", "Filtrar pelo CNPJ ou CPF do emitente")
	cmd.Flags().StringVar(&tomadorFlag, "tomador", "", "Filtrar pelo CNPJ ou CPF do tomador")
	cmd.Flags().StringVar(&nfeFlag, "nfe", "", "Filtrar pela chave de acesso de uma NF-e transportada")
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Filtrar pela chave de acesso (pode repetir)")

	return cmd
}
