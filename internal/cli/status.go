package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

var statusCNPJ string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Mostra um resumo da situação da empresa",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		result, err := application.Status(cmd.Context(), statusCNPJ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Status para: %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
		fmt.Printf("Ambiente: %s\n", result.Environment)
		fmt.Printf("Último NSU processado: %d\n", result.LastNSU)
		fmt.Println("\nMais estatísticas serão implementadas em breve...")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	statusCmd.MarkFlagRequired("cnpj")
}
