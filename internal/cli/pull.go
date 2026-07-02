package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newPullCommand(env CommandEnv) *cobra.Command {
	var cnpjFlag string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Sincroniza documentos fiscais da API ADN",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			result, err := application.Sync.Pull(cmd.Context(), app.PullInput{
				CNPJ: cnpjFlag,
			})
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Iniciando sincronização para %s (%s)\n",
				result.CompanyName, cnpj.Format(result.CNPJ))
			_, _ = fmt.Fprintf(out, "Sincronização concluída em %v\n", result.Duration.Round(1e6))
			_, _ = fmt.Fprintf(out, "Status: %s", result.Status)
			if result.StopReason != "" {
				_, _ = fmt.Fprintf(out, " (%s)", result.StopReason)
			}
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintf(out, "Último NSU consultado: %d\n", result.LastProcessedNSU)
			if result.LastFoundNSU != nil {
				_, _ = fmt.Fprintf(out, "Último NSU com documento: %d\n", *result.LastFoundNSU)
			} else {
				_, _ = fmt.Fprintf(out, "Último NSU com documento: -\n")
			}
			_, _ = fmt.Fprintf(out, "Documentos encontrados: %d | Erros: %d\n", result.DocumentsFound, result.Errors)

			return nil
		},
	}
	cmd.Flags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa para sincronizar")
	_ = cmd.MarkFlagRequired("cnpj")
	return cmd
}
