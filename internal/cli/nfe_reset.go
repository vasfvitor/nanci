package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// newNFeResetCmd removes the company's NF-e and resets its NF-e sync. Without
// --confirmar it only prints what would be removed.
func newNFeResetCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var confirmarFlag bool
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove as NF-e da empresa e reinicia a sincronização NF-e (simulação sem --confirmar)",
		Long: "Remove as NF-e, eventos e marcas de exportação da empresa, nos dois ambientes, e reinicia a " +
			"sincronização NF-e desde o NSU 0. Notas vistas por outra empresa continuam para ela. O histórico " +
			"das manifestações enviadas é mantido, e as manifestações registradas na SEFAZ não são afetadas.",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			if !confirmarFlag {
				result, err := application.NFe.PreviewReset(cmd.Context(), *cnpjFlag)
				if err != nil {
					return fmt.Errorf("erro: %w", err)
				}
				_, _ = fmt.Fprintf(out, "Redefinição de NF-e para: %s (%s), ambiente %s\n", result.CompanyName, cnpj.Format(result.CNPJ), result.Environment)
				printNFeResetCounts(out, result, "seriam removidos")
				_, _ = fmt.Fprintln(out, "\nNada foi alterado. Use --confirmar para redefinir.")
				return nil
			}

			result, err := application.NFe.Reset(cmd.Context(), *cnpjFlag)
			if err != nil {
				if errors.Is(err, app.ErrSyncRunning) {
					return fmt.Errorf("erro: %w; aguarde a sincronização atual terminar", err)
				}
				return fmt.Errorf("erro: %w", err)
			}
			_, _ = fmt.Fprintf(out, "NF-e redefinidas para: %s (%s), ambiente %s\n", result.CompanyName, cnpj.Format(result.CNPJ), result.Environment)
			printNFeResetCounts(out, result, "removidos")
			_, _ = fmt.Fprintln(out, "\nO próximo `nanci nfe pull` baixa a distribuição desde o NSU 0. O ambiente da empresa já pode ser alterado.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Remove de fato; sem esta flag nada é alterado")
	return cmd
}

func printNFeResetCounts(out io.Writer, result app.NFeResetResult, verb string) {
	_, _ = fmt.Fprintf(out, "  Notas da empresa (%s): %d\n", verb, result.CompanyDocuments)
	_, _ = fmt.Fprintf(out, "  Documentos que nenhuma outra empresa vê (%s): %d\n", verb, result.Documents)
	_, _ = fmt.Fprintf(out, "  Eventos (%s): %d\n", verb, result.Events)
	_, _ = fmt.Fprintf(out, "  Marcas de exportação (%s): %d\n", verb, result.ExportMarks)
	_, _ = fmt.Fprintf(out, "  Manifestações enviadas mantidas no histórico: %d\n", result.ManifestacoesKept)
	_, _ = fmt.Fprintln(out, "O cursor NF-e volta ao NSU 0; um bloqueio da SEFAZ em vigor continua valendo.")
	_, _ = fmt.Fprintln(out, "As manifestações registradas na SEFAZ não são afetadas.")
}
