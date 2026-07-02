// Package cli builds the nanci command tree.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/app"
)

func newInitCommand(env CommandEnv) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Inicializa o banco de dados e diretórios locais",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("erro ao inicializar: %w", err)
			}
			defer cleanup()

			dataDir, _ := app.ResolveRuntimeDataDir("")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pronto. Banco de dados criado/atualizado em: %s\n", dataDir)
			return nil
		},
	}
}
