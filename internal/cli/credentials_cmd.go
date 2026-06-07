package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

var (
	credentialLabel string
	credentialPath  string
	credentialEnv   string
	credentialID    string
)

var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "Gerencia credenciais reutilizáveis",
}

var credentialAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adiciona uma nova credencial",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		if err := application.AddCredential(context.Background(), app.AddCredentialInput{
			Label:       credentialLabel,
			CertPath:    credentialPath,
			Environment: credentialEnv,
		}); err != nil {
			return fmt.Errorf("erro ao adicionar credencial: %w", err)
		}
		fmt.Println("Credencial adicionada com sucesso.")
		return nil
	},
}

var credentialListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista todas as credenciais",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		credentials, err := application.ListCredentials(context.Background())
		if err != nil {
			return fmt.Errorf("erro ao listar credenciais: %w", err)
		}
		if len(credentials) == 0 {
			fmt.Println("Nenhuma credencial cadastrada.")
			return nil
		}

		fmt.Printf("%-36s %-18s %-15s %-20s %s\n", "ID", "Rótulo", "Ambiente", "CNPJ Proprietário", "Certificado")
		fmt.Println("----------------------------------------------------------------------------------------------------------------")
		for _, credential := range credentials {
			owner := credential.OwnerCNPJ
			if owner == "" {
				owner = "pendente"
			} else {
				owner = cnpj.Format(owner)
			}
			fmt.Printf("%-36s %-18s %-15s %-20s %s\n", credential.ID, credential.Label, credential.Environment, owner, credential.CertPath)
		}
		return nil
	},
}

var credentialUpdatePathCmd = &cobra.Command{
	Use:   "update-path",
	Short: "Atualiza o caminho do certificado de uma credencial",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		if err := application.UpdateCredentialPath(context.Background(), app.UpdateCredentialPathInput{
			CredentialID: credentialID,
			CertPath:     credentialPath,
		}); err != nil {
			return fmt.Errorf("erro ao atualizar credencial: %w", err)
		}
		fmt.Println("Caminho da credencial atualizado com sucesso.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(credentialCmd)
	credentialCmd.AddCommand(credentialAddCmd)
	credentialCmd.AddCommand(credentialListCmd)
	credentialCmd.AddCommand(credentialUpdatePathCmd)

	credentialAddCmd.Flags().StringVar(&credentialLabel, "label", "", "Rótulo da credencial")
	credentialAddCmd.Flags().StringVar(&credentialPath, "cert", "", "Caminho do certificado .pfx/.p12")
	credentialAddCmd.Flags().StringVar(&credentialEnv, "env", "producao_restrita", "Ambiente: producao ou producao_restrita")
	credentialAddCmd.MarkFlagRequired("cert")

	credentialUpdatePathCmd.Flags().StringVar(&credentialID, "credential-id", "", "ID da credencial")
	credentialUpdatePathCmd.Flags().StringVar(&credentialPath, "cert", "", "Novo caminho do certificado .pfx/.p12")
	credentialUpdatePathCmd.MarkFlagRequired("credential-id")
	credentialUpdatePathCmd.MarkFlagRequired("cert")
}
