// Package cli builds the nanci command tree.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInitCommand(env CommandEnv) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Inicializa o banco de dados e diretórios locais",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("erro ao inicializar: %w", err)
			}
			defer cleanup()

			application.Log.Info("Ambiente inicializado com sucesso!", "data_dir", application.DataDir)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pronto. Banco de dados criado/atualizado em: %s\n", application.DataDir)
			return nil
		},
	}
}
