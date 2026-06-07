package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa o banco de dados e diretórios locais",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("erro ao inicializar: %w", err)
		}
		defer application.Close()

		application.Log.Info("Ambiente inicializado com sucesso!", "data_dir", application.DataDir)
		fmt.Printf("Pronto. Banco de dados criado/atualizado em: %s\n", application.DataDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
