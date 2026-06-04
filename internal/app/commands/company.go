package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
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
	Run: func(cmd *cobra.Command, args []string) {
		app, err := initApp()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
			os.Exit(1)
		}
		defer app.Close()

		if err := cnpj.Validate(companyCNPJ); err != nil {
			fmt.Fprintf(os.Stderr, "CNPJ inválido: %v\n", err)
			os.Exit(1)
		}

		cleanedCNPJ := cnpj.Clean(companyCNPJ)
		root, _ := cnpj.Root(cleanedCNPJ)

		// Check if certificate file exists
		if _, err := os.Stat(companyCert); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Arquivo de certificado não encontrado: %s\n", companyCert)
			os.Exit(1)
		}

		company := &nfse.Company{
			ID:          uuid.NewString(),
			CNPJ:        cleanedCNPJ,
			CNPJRoot:    root,
			Name:        companyName,
			CertPath:    companyCert,
			Environment: companyEnv,
		}

		ctx := context.Background()
		if err := app.Store.CreateCompany(ctx, company); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao salvar empresa: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Empresa '%s' (%s) adicionada com sucesso.\n", company.Name, cnpj.Format(company.CNPJ))
	},
}

var companyListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista todas as empresas",
	Run: func(cmd *cobra.Command, args []string) {
		app, err := initApp()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar: %v\n", err)
			os.Exit(1)
		}
		defer app.Close()

		ctx := context.Background()
		companies, err := app.Store.ListCompanies(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao listar empresas: %v\n", err)
			os.Exit(1)
		}

		if len(companies) == 0 {
			fmt.Println("Nenhuma empresa cadastrada.")
			return
		}

		fmt.Printf("%-20s %-30s %-15s %s\n", "CNPJ", "Nome", "Ambiente", "Último NSU")
		fmt.Println("---------------------------------------------------------------------------------")
		for _, c := range companies {
			fmt.Printf("%-20s %-30s %-15s %d\n", cnpj.Format(c.CNPJ), c.Name, c.Environment, c.LastNSU)
		}
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
