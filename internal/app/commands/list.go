package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/store"
)

var (
	listCNPJ       string
	listCompetence string
	listDirection  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista documentos fiscais sincronizados",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := initApp()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
			os.Exit(1)
		}
		defer app.Close()

		if err := cnpj.Validate(listCNPJ); err != nil {
			fmt.Fprintf(os.Stderr, "CNPJ inválido: %v\n", err)
			os.Exit(1)
		}

		cleanedCNPJ := cnpj.Clean(listCNPJ)
		ctx := cmd.Context()

		company, err := app.Store.GetCompany(ctx, cleanedCNPJ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao buscar empresa: %v\n", err)
			os.Exit(1)
		}
		if company == nil {
			fmt.Fprintf(os.Stderr, "Empresa não encontrada para o CNPJ %s\n", cnpj.Format(cleanedCNPJ))
			os.Exit(1)
		}

		filter := store.DocumentFilter{
			Competence: listCompetence,
			Direction:  listDirection,
		}

		docs, err := app.Store.ListDocuments(ctx, company.ID, filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao listar documentos: %v\n", err)
			os.Exit(1)
		}

		if len(docs) == 0 {
			fmt.Println("Nenhum documento encontrado.")
			return
		}

		// Configure tabwriter
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "EMISSÃO\tCHAVE DE ACESSO\tDIREÇÃO\tPRESTADOR\tTOMADOR\tVALOR (R$)\tISS\tIRRF")
		fmt.Fprintln(w, "-------\t---------------\t-------\t---------\t-------\t----------\t---\t----")

		for _, d := range docs {
			issueStr := ""
			if !d.IssueDate.IsZero() {
				issueStr = d.IssueDate.Format("2006-02-01")
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.2f\t%.2f\t%.2f\n",
				issueStr,
				d.ChaveAcesso,
				d.Direction,
				cnpj.Format(d.PrestadorCNPJ),
				cnpj.Format(d.TomadorCNPJ),
				d.ServiceValue,
				d.ISSValue,
				d.IRRFValue,
			)
		}

		w.Flush()
		fmt.Printf("\nTotal de %d documento(s) listado(s).\n", len(docs))
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	listCmd.Flags().StringVarP(&listCompetence, "competencia", "m", "", "Filtrar por competência (ex: 2026-06)")
	listCmd.Flags().StringVarP(&listDirection, "direcao", "d", "", "Filtrar por direção (tomada, prestada, intermediario)")

	listCmd.MarkFlagRequired("cnpj")
}
