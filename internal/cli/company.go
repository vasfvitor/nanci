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
	companyCNPJ            string
	companyName            string
	companyCert            string
	companyEnv             string
	companyCredentialID    string
	companyCredentialLabel string
	assignCredentialID     string
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
			CNPJ:            companyCNPJ,
			Name:            companyName,
			CredentialID:    companyCredentialID,
			CredentialLabel: companyCredentialLabel,
			CertPath:        companyCert,
			Environment:     companyEnv,
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

		fmt.Printf("%-20s %-24s %-18s %-15s %s\n", "CNPJ", "Nome", "Credencial", "Ambiente", "Último NSU")
		fmt.Println("------------------------------------------------------------------------------------------------")
		for _, c := range companies {
			fmt.Printf("%-20s %-24s %-18s %-15s %d\n", cnpj.Format(c.CNPJ), c.Name, c.CredentialLabel, c.Environment, c.LastNSU)
		}
		return nil
	},
}

var companyAssignCredentialCmd = &cobra.Command{
	Use:   "assign-credential",
	Short: "Atribui uma credencial existente a uma empresa",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		if err := application.AssignCredentialToCompany(context.Background(), app.AssignCredentialInput{
			CompanyCNPJ:  companyCNPJ,
			CredentialID: assignCredentialID,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao atribuir credencial: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Credencial atribuída com sucesso.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(companyCmd)
	companyCmd.AddCommand(companyAddCmd)
	companyCmd.AddCommand(companyListCmd)
	companyCmd.AddCommand(companyAssignCredentialCmd)

	companyAddCmd.Flags().StringVarP(&companyCNPJ, "cnpj", "c", "", "CNPJ da empresa (numérico com DV válido; alfanumérico ainda não suportado)")
	companyAddCmd.Flags().StringVarP(&companyName, "name", "n", "", "Nome ou Razão Social")
	companyAddCmd.Flags().StringVarP(&companyCert, "cert", "p", "", "Caminho para o certificado .pfx/.p12")
	companyAddCmd.Flags().StringVar(&companyCredentialID, "credential-id", "", "ID de uma credencial existente")
	companyAddCmd.Flags().StringVar(&companyCredentialLabel, "credential-label", "", "Rótulo da nova credencial quando criada inline")
	companyAddCmd.Flags().StringVarP(&companyEnv, "env", "e", "producao_restrita", "Ambiente: producao ou producao_restrita")

	companyAddCmd.MarkFlagRequired("cnpj")
	companyAddCmd.MarkFlagRequired("name")

	companyAssignCredentialCmd.Flags().StringVarP(&companyCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	companyAssignCredentialCmd.Flags().StringVar(&assignCredentialID, "credential-id", "", "ID da credencial")
	companyAssignCredentialCmd.MarkFlagRequired("cnpj")
	companyAssignCredentialCmd.MarkFlagRequired("credential-id")
}
