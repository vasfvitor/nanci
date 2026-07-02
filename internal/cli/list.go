package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newListCommand(env CommandEnv) *cobra.Command {
	var (
		cnpjFlag       string
		competenceFlag string
		directionFlag  string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista documentos fiscais sincronizados",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			docs, err := application.Documents.ListDocuments(cmd.Context(), app.ListInput{
				CNPJ:       cnpjFlag,
				Competence: competenceFlag,
				Direction:  directionFlag,
			})
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			if len(docs) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Nenhum documento encontrado.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "EMISSÃO\tCHAVE DE ACESSO\tDIREÇÃO\tVISIBILIDADE\tPRESTADOR\tTOMADOR\tVALOR (R$)\tISS\tIRRF")
			_, _ = fmt.Fprintln(w, "-------\t---------------\t-------\t------------\t---------\t-------\t----------\t---\t----")

			for _, d := range docs {
				issueStr := ""
				if !d.IssueDate.IsZero() {
					issueStr = d.IssueDate.Format("2006-01-02")
				}
				_, _ = fmt.Fprintf(
					w, "%s\t%s\t%s\t%s\t%s\t%s\t%.2f\t%.2f\t%.2f\n",
					issueStr,
					d.ChaveAcesso,
					d.CompanyRole,
					d.VisibilityReason,
					cnpj.Format(d.PrestadorCNPJ),
					cnpj.Format(d.TomadorCNPJ),
					float64(d.ServiceValue)/100.0,
					float64(d.ISSValue)/100.0,
					float64(d.IRRFValue)/100.0,
				)
			}

			_ = w.Flush()
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal de %d documento(s) listado(s).\n", len(docs))
			return nil
		},
	}
	cmd.Flags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa")
	cmd.Flags().StringVarP(&competenceFlag, "competencia", "m", "", "Filtrar por competência (ex: 2026-06)")
	cmd.Flags().StringVarP(&directionFlag, "direcao", "d", "", "Filtrar por direção (tomada, prestada, intermediario)")
	_ = cmd.MarkFlagRequired("cnpj")
	return cmd
}
