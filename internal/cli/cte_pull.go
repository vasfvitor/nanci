package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newCTePullCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Baixa CT-e e eventos da distribuição DF-e da SEFAZ",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			result, err := application.CTe.Pull(cmd.Context(), *cnpjFlag)
			if err != nil {
				return pullError(out, err)
			}

			printCTePullResult(out, result)
			return nil
		},
	}
}

func printCTePullResult(out io.Writer, result app.CTePullResult) {
	_, _ = fmt.Fprintf(out, "Sincronização de CT-e para %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
	_, _ = fmt.Fprintf(out, "Concluída em %v\n", result.Duration.Round(1e6))
	_, _ = fmt.Fprintf(out, "Status: %s", result.Status)
	if result.StopReason != "" {
		_, _ = fmt.Fprintf(out, " (%s)", result.StopReason)
	}
	_, _ = fmt.Fprintln(out)

	_, _ = fmt.Fprintf(out, "NSU consultado: %s / máximo: %s\n", formatNSU(result.LastNSU), formatMaxNSU(result.MaxNSU))
	_, _ = fmt.Fprintf(out, "CT-e salvos: %d | Eventos salvos: %d | Erros: %d\n",
		result.DocumentsSaved, result.EventsSaved, result.Errors)
	_, _ = fmt.Fprintf(out, "Consultas na última hora: %d/%d\n", result.RequestsLastHour, result.RequestBudget)
	if result.NextAllowedAt != nil {
		_, _ = fmt.Fprintf(out, "Próxima consulta permitida após: %s\n", formatDateTime(*result.NextAllowedAt))
	}
}
