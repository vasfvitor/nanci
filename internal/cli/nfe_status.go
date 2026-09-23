package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

func newNFeStatusCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Mostra a situação da sincronização e das manifestações de NF-e",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			result, err := application.NFe.Status(cmd.Context(), *cnpjFlag)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Status NF-e para: %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
			_, _ = fmt.Fprintf(out, "Ambiente: %s (tpAmb %s) | UF: %s\n", ambienteLabel(result.TpAmb), result.TpAmb, dashIfEmpty(result.UF))

			_, _ = fmt.Fprintln(out, "\nSincronização:")
			maxNSU := "-"
			if result.MaxNSU > 0 {
				maxNSU = formatNSU(result.MaxNSU)
			}
			_, _ = fmt.Fprintf(out, "  Último NSU: %s / máximo: %s\n", formatNSU(result.LastNSU), maxNSU)
			if result.LastSyncAt != nil {
				_, _ = fmt.Fprintf(out, "  Última sincronização: %s\n", result.LastSyncAt.Local().Format("2006-01-02 15:04:05"))
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

			_, _ = fmt.Fprintln(out, "\nDocumentos:")
			_, _ = fmt.Fprintf(out, "  Como destinatária: %d\n", result.TotalDestinatario)
			_, _ = fmt.Fprintf(out, "  Como emitente: %d\n", result.TotalEmitente)
			_, _ = fmt.Fprintf(out, "  Outros papéis: %d\n", result.TotalOutros)
			_, _ = fmt.Fprintf(out, "  Completas: %d | Resumos: %d\n", result.TotalCompletas, result.TotalResumos)

			_, _ = fmt.Fprintln(out, "\nManifestação:")
			_, _ = fmt.Fprintf(out, "  Sem ciência: %d (%d com ciência atrasada)\n", result.PendingCiencia, result.CienciaOverdue)
			_, _ = fmt.Fprintf(out, "  Sem manifestação conclusiva: %d\n", result.PendingConclusiva)
			return nil
		},
	}
}

// ambienteLabel names the SEFAZ environment of tpAmb.
func ambienteLabel(tpAmb string) string {
	if tpAmb == sefaz.TpAmbProducao {
		return "Produção"
	}
	return "Homologação"
}
