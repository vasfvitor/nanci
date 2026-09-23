package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newNFePendentesCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var vencendoEmFlag int
	cmd := &cobra.Command{
		Use:   "pendentes",
		Short: "Lista as NF-e recebidas ainda sem manifestação conclusiva",
		RunE: func(cmd *cobra.Command, args []string) error {
			if vencendoEmFlag < 0 {
				return errors.New("--vencendo-em não pode ser negativo")
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			pending, err := application.NFe.ListPendingManifestations(cmd.Context(), app.NFePendingInput{
				CNPJ:          *cnpjFlag,
				DueWithinDays: vencendoEmFlag,
			})
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			if len(pending) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma manifestação pendente.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "CHAVE DE ACESSO\tEMITENTE\tEMISSÃO\tVALOR (R$)\tMANIFESTAÇÃO\tCIÊNCIA ATÉ\tCONCLUSIVA ATÉ\tDIAS\tALERTA")
			_, _ = fmt.Fprintln(w, "---------------\t--------\t-------\t----------\t------------\t-----------\t--------------\t----\t------")
			for _, p := range pending {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
					p.ChaveAcesso,
					cnpj.Format(p.EmitenteCNPJ),
					formatNFeDate(p.IssueDate),
					p.TotalValue.FormatBRL(),
					p.Manifestacao,
					formatNFeDate(p.CienciaDue),
					formatNFeDate(p.ConclusiveDue),
					p.DaysLeft,
					pendingAlert(p),
				)
			}
			_ = w.Flush()
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal de %d nota(s) pendente(s).\n", len(pending))
			return nil
		},
	}
	cmd.Flags().IntVar(&vencendoEmFlag, "vencendo-em", 0, "Mostrar apenas as NF-e cujo prazo conclusivo vence em até N dias (0 mostra todas)")
	return cmd
}

// pendingAlert is the ALERTA column of `nfe pendentes`.
func pendingAlert(p app.NFePendingManifestation) string {
	switch {
	case p.TacitlyConfirmed:
		return "confirmada tacitamente (prazo expirado)"
	case p.CienciaOverdue:
		return "ciência atrasada"
	default:
		return "-"
	}
}
