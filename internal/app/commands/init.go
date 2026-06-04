package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa o banco de dados e diretórios locais",
	Run: func(cmd *cobra.Command, args []string) {
		application, err := initApp()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
			os.Exit(1)
		}
		defer application.Close()

		application.Log.Info("Ambiente inicializado com sucesso!", "data_dir", application.DataDir)
		fmt.Printf("Pronto. Banco de dados criado/atualizado em: %s\n", application.DataDir)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
