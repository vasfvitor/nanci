package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newCTeStatusCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Mostra a situação da sincronização e os totais de CT-e",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			result, err := application.CTe.Status(cmd.Context(), *cnpjFlag)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Status CT-e para: %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
			_, _ = fmt.Fprintf(out, "Ambiente: %s (tpAmb %s) | UF: %s\n", ambienteLabel(result.TpAmb), result.TpAmb, dashIfEmpty(result.UF))

			_, _ = fmt.Fprintln(out, "\nSincronização:")
			_, _ = fmt.Fprintf(out, "  Último NSU: %s / máximo: %s\n", formatNSU(result.LastNSU), formatMaxNSU(result.MaxNSU))
			if result.LastSyncAt != nil {
				_, _ = fmt.Fprintf(out, "  Última sincronização: %s\n", formatNFeDateTime(*result.LastSyncAt))
			}
			if result.LastRunStatus != "" {
				_, _ = fmt.Fprintf(out, "  Última execução: %s", result.LastRunStatus)
				if result.LastRunStopReason != "" {
					_, _ = fmt.Fprintf(out, " (%s)", result.LastRunStopReason)
				}
				_, _ = fmt.Fprintln(out)
			}
			if result.InitialSyncDoneAt != nil {
				_, _ = fmt.Fprintf(out, "  Carga inicial concluída em: %s\n", formatNFeDateTime(*result.InitialSyncDoneAt))
			} else {
				_, _ = fmt.Fprintln(out, "  Carga inicial: pendente")
			}
			_, _ = fmt.Fprintf(out, "  Consultas na última hora: %d/%d\n", result.RequestsLastHour, result.RequestBudget)
			if result.NextAllowedAt != nil {
				_, _ = fmt.Fprintf(out, "  Próxima consulta permitida após: %s (%s)\n", formatNFeDateTime(*result.NextAllowedAt), result.BlockedReason)
			}

			_, _ = fmt.Fprintln(out, "\nDocumentos, pelo papel principal:")
			_, _ = fmt.Fprintf(out, "  Como tomadora: %d\n", result.TotalTomador)
			_, _ = fmt.Fprintf(out, "  Como destinatária: %d\n", result.TotalDestinatario)
			_, _ = fmt.Fprintf(out, "  Como remetente: %d\n", result.TotalRemetente)
			_, _ = fmt.Fprintf(out, "  Outros papéis: %d\n", result.TotalOutros)
			return nil
		},
	}
}
