package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

func newCredentialCommand(env CommandEnv) *cobra.Command {
	credentialCmd := &cobra.Command{
		Use:   "credential",
		Short: "Gerencia credenciais reutilizáveis",
	}
	credentialCmd.AddCommand(newCredentialAddCmd(env))
	credentialCmd.AddCommand(newCredentialListCmd(env))
	credentialCmd.AddCommand(newCredentialUpdatePathCmd(env))
	return credentialCmd
}

func newCredentialAddCmd(env CommandEnv) *cobra.Command {
	var (
		label string
		path  string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adiciona uma nova credencial",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			if err := application.Credentials.AddCredential(cmd.Context(), credential.AddCredentialInput{
				Label:    label,
				CertPath: path,
			}); err != nil {
				return fmt.Errorf("erro ao adicionar credencial: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Credencial adicionada com sucesso.")
			return nil
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "Rótulo da credencial")
	cmd.Flags().StringVar(&path, "cert", "", "Caminho do certificado .pfx/.p12")
	_ = cmd.MarkFlagRequired("cert")
	return cmd
}

func newCredentialListCmd(env CommandEnv) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista todas as credenciais",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			credentials, err := application.Credentials.ListCredentials(cmd.Context())
			if err != nil {
				return fmt.Errorf("erro ao listar credenciais: %w", err)
			}
			if len(credentials) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma credencial cadastrada.")
				return nil
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "%-36s %-18s %-20s %s\n", "ID", "Rótulo", "CNPJ Proprietário", "Certificado")
			_, _ = fmt.Fprintln(out, "----------------------------------------------------------------------------------------------------------------")
			for _, credential := range credentials {
				owner := credential.OwnerCNPJ
				if owner == "" {
					owner = "pendente"
				} else {
					owner = cnpj.Format(owner)
				}
				_, _ = fmt.Fprintf(out, "%-36s %-18s %-20s %s\n", credential.ID, credential.Label, owner, credential.CertPath)
			}
			return nil
		},
	}
}

func newCredentialUpdatePathCmd(env CommandEnv) *cobra.Command {
	var (
		id   string
		path string
	)
	cmd := &cobra.Command{
		Use:   "update-path",
		Short: "Atualiza o caminho do certificado de uma credencial",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			if err := application.Credentials.UpdateCredentialPath(cmd.Context(), credential.UpdateCredentialPathInput{
				CredentialID: id,
				CertPath:     path,
			}); err != nil {
				return fmt.Errorf("erro ao atualizar credencial: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Caminho da credencial atualizado com sucesso.")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "credential-id", "", "ID da credencial")
	cmd.Flags().StringVar(&path, "cert", "", "Novo caminho do certificado .pfx/.p12")
	_ = cmd.MarkFlagRequired("credential-id")
	_ = cmd.MarkFlagRequired("cert")
	return cmd
}
