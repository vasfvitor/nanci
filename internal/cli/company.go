package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

var (
	companyCNPJ string
	companyName string
	companyCert string
	companyEnv  string
)

var companyCmd = &cobra.Command{
	Use:   "company",
	Short: "Gerencia empresas (contribuintes)",
}

var companyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adiciona uma nova empresa",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		if err := application.AddCompany(context.Background(), app.AddCompanyInput{
			CNPJ:        companyCNPJ,
			Name:        companyName,
			CertPath:    companyCert,
			Environment: companyEnv,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao adicionar empresa: %v\n", err)
			os.Exit(1)
		}

		cleanedCNPJ := cnpj.Clean(companyCNPJ)
		fmt.Printf("Empresa '%s' (%s) adicionada com sucesso.\n", companyName, cnpj.Format(cleanedCNPJ))
		return nil
	},
}

var companyListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista todas as empresas",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		companies, err := application.ListCompanies(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao listar empresas: %v\n", err)
			os.Exit(1)
		}

		if len(companies) == 0 {
			fmt.Println("Nenhuma empresa cadastrada.")
			return nil
		}

		fmt.Printf("%-20s %-30s %-15s %s\n", "CNPJ", "Nome", "Ambiente", "Último NSU")
		fmt.Println("---------------------------------------------------------------------------------")
		for _, c := range companies {
			fmt.Printf("%-20s %-30s %-15s %d\n", cnpj.Format(c.CNPJ), c.Name, c.Environment, c.LastNSU)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(companyCmd)
	companyCmd.AddCommand(companyAddCmd)
	companyCmd.AddCommand(companyListCmd)

	companyAddCmd.Flags().StringVarP(&companyCNPJ, "cnpj", "c", "", "CNPJ da empresa (somente números ou formato RFB)")
	companyAddCmd.Flags().StringVarP(&companyName, "name", "n", "", "Nome ou Razão Social")
	companyAddCmd.Flags().StringVarP(&companyCert, "cert", "p", "", "Caminho para o certificado .pfx/.p12")
	companyAddCmd.Flags().StringVarP(&companyEnv, "env", "e", "producao_restrita", "Ambiente: producao ou producao_restrita")

	companyAddCmd.MarkFlagRequired("cnpj")
	companyAddCmd.MarkFlagRequired("name")
	companyAddCmd.MarkFlagRequired("cert")
}
