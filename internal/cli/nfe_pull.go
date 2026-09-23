package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/sync"
)

func newNFePullCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Baixa NF-e, resumos e eventos da distribuição DF-e da SEFAZ",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			result, err := application.NFe.Pull(cmd.Context(), *cnpjFlag)
			if err != nil {
				var blocked *sync.BlockedError
				if errors.As(err, &blocked) {
					_, _ = fmt.Fprintf(out, "Próxima consulta permitida após: %s\n", formatNFeDateTime(blocked.Until))
				}
				if errors.Is(err, app.ErrSyncRunning) {
					return fmt.Errorf("erro: %w; aguarde a sincronização atual terminar", err)
				}
				return fmt.Errorf("erro: %w", err)
			}

			printNFePullResult(out, result)
			return nil
		},
	}
}

func printNFePullResult(out io.Writer, result app.NFePullResult) {
	_, _ = fmt.Fprintf(out, "Sincronização de NF-e para %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
	_, _ = fmt.Fprintf(out, "Concluída em %v\n", result.Duration.Round(1e6))
	_, _ = fmt.Fprintf(out, "Status: %s", result.Status)
	if result.StopReason != "" {
		_, _ = fmt.Fprintf(out, " (%s)", result.StopReason)
	}
	_, _ = fmt.Fprintln(out)

	_, _ = fmt.Fprintf(out, "NSU consultado: %s / máximo: %s\n", formatNSU(result.UltNSU), formatMaxNSU(result.MaxNSU))
	_, _ = fmt.Fprintf(out, "NF-e completas salvas: %d | Resumos salvos: %d | Eventos salvos: %d | Erros: %d\n",
		result.CompletasSaved, result.ResumosSaved, result.EventsSaved, result.Errors)
	_, _ = fmt.Fprintf(out, "Consultas na última hora: %d/%d\n", result.RequestsLastHour, result.RequestBudget)
	if result.NextAllowedAt != nil {
		_, _ = fmt.Fprintf(out, "Próxima consulta permitida após: %s\n", formatNFeDateTime(*result.NextAllowedAt))
	}
}
