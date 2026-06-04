package commands

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
	Run: func(cmd *cobra.Command, args []string) {
		app, err := initApp()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
			os.Exit(1)
		}
		defer app.Close()

		if err := cnpj.Validate(statusCNPJ); err != nil {
			fmt.Fprintf(os.Stderr, "CNPJ inválido: %v\n", err)
			os.Exit(1)
		}

		cleanedCNPJ := cnpj.Clean(statusCNPJ)

		// Fetch the company
		ctx := cmd.Context()
		company, err := app.Store.GetCompany(ctx, cleanedCNPJ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao buscar empresa: %v\n", err)
			os.Exit(1)
		}

		if company == nil {
			fmt.Fprintf(os.Stderr, "Empresa não encontrada para o CNPJ %s\n", cnpj.Format(cleanedCNPJ))
			os.Exit(1)
		}

		fmt.Printf("Status para: %s (%s)\n", company.Name, cnpj.Format(company.CNPJ))
		fmt.Printf("Ambiente: %s\n", company.Environment)
		fmt.Printf("Último NSU processado: %d\n", company.LastNSU)
		fmt.Println("\nMais estatísticas serão implementadas em breve...")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	statusCmd.MarkFlagRequired("cnpj")
}
