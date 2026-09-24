package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// newCTeResetCmd removes the company's CT-e and resets its CT-e sync. Without
// --confirmar it only prints what would be removed.
func newCTeResetCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var confirmarFlag bool
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove os CT-e da empresa e reinicia a sincronização CT-e (simulação sem --confirmar)",
		Long: "Remove os CT-e, eventos e marcas de exportação da empresa, nos dois ambientes, e reinicia a " +
			"sincronização CT-e desde o NSU 0. Documentos vistos por outra empresa continuam para ela.",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			if !confirmarFlag {
				result, err := application.CTe.PreviewReset(cmd.Context(), *cnpjFlag)
				if err != nil {
					return fmt.Errorf("erro: %w", err)
				}
				_, _ = fmt.Fprintf(out, "Redefinição de CT-e para: %s (%s), ambiente %s\n", result.CompanyName, cnpj.Format(result.CNPJ), result.Environment)
				printCTeResetCounts(out, result, "seriam removidos")
				_, _ = fmt.Fprintln(out, "\nNada foi alterado. Use --confirmar para redefinir.")
				return nil
			}

			result, err := application.CTe.Reset(cmd.Context(), *cnpjFlag)
			if err != nil {
				if errors.Is(err, app.ErrSyncRunning) {
					return fmt.Errorf("erro: %w; aguarde a sincronização atual terminar", err)
				}
				return fmt.Errorf("erro: %w", err)
			}
			_, _ = fmt.Fprintf(out, "CT-e redefinidos para: %s (%s), ambiente %s\n", result.CompanyName, cnpj.Format(result.CNPJ), result.Environment)
			printCTeResetCounts(out, result, "removidos")
			_, _ = fmt.Fprintln(out, "\nO próximo `nanci cte pull` baixa a distribuição desde o NSU 0.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Remove de fato; sem esta flag nada é alterado")
	return cmd
}

func printCTeResetCounts(out io.Writer, result app.CTeResetResult, verb string) {
	_, _ = fmt.Fprintf(out, "  CT-e da empresa (%s): %d\n", verb, result.CompanyDocuments)
	_, _ = fmt.Fprintf(out, "  Documentos que nenhuma outra empresa vê (%s): %d\n", verb, result.Documents)
	_, _ = fmt.Fprintf(out, "  Eventos (%s): %d\n", verb, result.Events)
	_, _ = fmt.Fprintf(out, "  Marcas de exportação (%s): %d\n", verb, result.ExportMarks)
	_, _ = fmt.Fprintln(out, "O cursor CT-e volta ao NSU 0; um bloqueio da SEFAZ em vigor continua valendo.")
}
