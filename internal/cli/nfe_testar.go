package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNFeTestarConexaoCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "testar-conexao",
		Short: "Testa o certificado e a conexão TLS com a SEFAZ sem consumir consultas",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			result, err := application.NFe.TestConnection(cmd.Context(), *cnpjFlag)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			return printConnectionTest(cmd.OutOrStdout(), result)
		},
	}
}
