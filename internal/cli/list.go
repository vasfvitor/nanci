package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

var (
	listCNPJ       string
	listCompetence string
	listDirection  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista documentos fiscais sincronizados",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		docs, err := application.ListDocuments(cmd.Context(), app.ListInput{
			CNPJ:       listCNPJ,
			Competence: listCompetence,
			Direction:  listDirection,
		})
		if err != nil {
			return fmt.Errorf("erro: %w", err)
		}

		if len(docs) == 0 {
			fmt.Println("Nenhum documento encontrado.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "EMISSÃO\tCHAVE DE ACESSO\tDIREÇÃO\tVISIBILIDADE\tPRESTADOR\tTOMADOR\tVALOR (R$)\tISS\tIRRF")
		_, _ = fmt.Fprintln(w, "-------\t---------------\t-------\t------------\t---------\t-------\t----------\t---\t----")

		for _, d := range docs {
			issueStr := ""
			if !d.IssueDate.IsZero() {
				issueStr = d.IssueDate.Format("2006-01-02")
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%.2f\t%.2f\t%.2f\n",
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
		fmt.Printf("\nTotal de %d documento(s) listado(s).\n", len(docs))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	listCmd.Flags().StringVarP(&listCompetence, "competencia", "m", "", "Filtrar por competência (ex: 2026-06)")
	listCmd.Flags().StringVarP(&listDirection, "direcao", "d", "", "Filtrar por direção (tomada, prestada, intermediario)")
	_ = listCmd.MarkFlagRequired("cnpj")
}
