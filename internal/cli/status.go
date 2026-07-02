package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newStatusCommand(env CommandEnv) *cobra.Command {
	var cnpjFlag string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Mostra um resumo da situação da empresa",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			result, err := application.Status.Status(cmd.Context(), cnpjFlag)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Status para: %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
			_, _ = fmt.Fprintf(out, "Ambiente: %s\n", result.Environment)
			_, _ = fmt.Fprintf(out, "CNPJ consultado: %s\n", cnpj.Format(result.ConsultationCNPJ))
			_, _ = fmt.Fprintf(out, "Último NSU consultado: %d\n", result.LastProcessedNSU)
			if result.LastFoundNSU != nil {
				_, _ = fmt.Fprintf(out, "Último NSU com documento: %d\n", *result.LastFoundNSU)
			} else {
				_, _ = fmt.Fprintf(out, "Último NSU com documento: -\n")
			}
			if result.LastSyncAt != nil {
				_, _ = fmt.Fprintf(out, "Última sincronização: %s\n", result.LastSyncAt.Format("2006-01-02 15:04:05"))
			}
			if result.LastRunStatus != "" {
				_, _ = fmt.Fprintf(out, "Última execução: %s", result.LastRunStatus)
				if result.LastRunStopReason != "" {
					_, _ = fmt.Fprintf(out, " (%s)", result.LastRunStopReason)
				}
				_, _ = fmt.Fprintln(out)
			}
			_, _ = fmt.Fprintln(out, "\nEstatísticas de Documentos:")
			_, _ = fmt.Fprintf(out, "  Notas Emitidas (Prestadas): %d\n", result.TotalEmitidas)
			_, _ = fmt.Fprintf(out, "  Notas Tomadas: %d\n", result.TotalTomadas)
			return nil
		},
	}
	cmd.Flags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa")
	_ = cmd.MarkFlagRequired("cnpj")
	return cmd
}
