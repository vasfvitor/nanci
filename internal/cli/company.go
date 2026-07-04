package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/company"
	cnpjpkg "github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// newCompanyCommand builds the `company` subcommand tree from the given env.
// All flag vars are local to this constructor; the four leaves (add, list,
// assign-credential, ...) share the env.AppFactory seam to reach the app.
func newCompanyCommand(env CommandEnv) *cobra.Command {
	companyCmd := &cobra.Command{
		Use:   "company",
		Short: "Gerencia empresas (contribuintes)",
	}

	companyCmd.AddCommand(newCompanyAddCmd(env))
	companyCmd.AddCommand(newCompanyListCmd(env))
	companyCmd.AddCommand(newCompanyAssignCredentialCmd(env))
	return companyCmd
}

func newCompanyAddCmd(env CommandEnv) *cobra.Command {
	var (
		cnpj            string
		name            string
		cert            string
		envName         string
		credentialID    string
		credentialLabel string
		syncStartPolicy string
		syncStartDate   string
		last12Months    bool
		last5Years      bool
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adiciona uma nova empresa",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			environment, err := nfse.ParseEnvironment(envName)
			if err != nil {
				return fmt.Errorf("erro no ambiente: %w", err)
			}
			rawSyncStartPolicy, rawSyncStartDate := resolveCompanySyncStartFlags(syncStartPolicy, syncStartDate, last12Months, last5Years)
			policy, date, err := company.ParseSyncStartPolicyInput(rawSyncStartPolicy, rawSyncStartDate)
			if err != nil {
				return fmt.Errorf("erro na politica de sincronização: %w", err)
			}

			if err := application.Companies.AddCompany(cmd.Context(), company.AddCompanyInput{
				CNPJ:            cnpj,
				Name:            name,
				CredentialID:    credentialID,
				CredentialLabel: credentialLabel,
				CertPath:        cert,
				Environment:     environment,
				SyncStartPolicy: policy,
				SyncStartDate:   date,
			}); err != nil {
				return fmt.Errorf("erro ao adicionar empresa: %w", err)
			}

			cleanedCNPJ := cnpjpkg.Clean(cnpj)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Empresa '%s' (%s) adicionada com sucesso.\n", name, cnpjpkg.Format(cleanedCNPJ))
			return nil
		},
	}
	cmd.Flags().StringVarP(&cnpj, "cnpj", "c", "", "CNPJ da empresa (numérico com DV válido; alfanumérico ainda não suportado)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Nome ou Razão Social")
	cmd.Flags().StringVarP(&cert, "cert", "p", "", "Caminho para o certificado .pfx/.p12")
	cmd.Flags().StringVar(&credentialID, "credential-id", "", "ID de uma credencial existente")
	cmd.Flags().StringVar(&credentialLabel, "credential-label", "", "Rótulo da nova credencial quando criada inline")
	cmd.Flags().StringVarP(&envName, "env", "e", "producao_restrita", "Ambiente: producao ou producao_restrita")
	cmd.Flags().StringVar(&syncStartPolicy, "sync-start-policy", "from_now", "Política inicial: all, since_date ou from_now")
	cmd.Flags().StringVar(&syncStartDate, "sync-start-date", "", "Data de corte inicial YYYY-MM-DD para since_date")
	cmd.Flags().BoolVar(&last12Months, "last-12-months", false, "Atalho para importar somente os últimos 12 meses no primeiro sync")
	cmd.Flags().BoolVar(&last5Years, "last-5-years", false, "Atalho para importar somente os últimos 5 anos no primeiro sync")
	_ = cmd.MarkFlagRequired("cnpj")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newCompanyListCmd(env CommandEnv) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista todas as empresas",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			companies, err := application.Companies.ListCompanies(cmd.Context())
			if err != nil {
				return fmt.Errorf("erro ao listar empresas: %w", err)
			}

			if len(companies) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma empresa cadastrada.")
				return nil
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "%-20s %-24s %-18s %-15s\n", "CNPJ", "Nome", "Credencial", "Ambiente")
			_, _ = fmt.Fprintln(out, "------------------------------------------------------------------------------------------------")
			for _, c := range companies {
				_, _ = fmt.Fprintf(out, "%-20s %-24s %-18s %-15s\n", cnpjpkg.Format(c.CNPJ), c.Name, c.CredentialLabel, c.Environment)
			}
			return nil
		},
	}
	return cmd
}

func newCompanyAssignCredentialCmd(env CommandEnv) *cobra.Command {
	var (
		cnpj         string
		credentialID string
	)
	cmd := &cobra.Command{
		Use:   "assign-credential",
		Short: "Atribui uma credencial existente a uma empresa",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			if err := application.Companies.AssignCredentialToCompany(cmd.Context(), company.AssignCredentialInput{
				CompanyCNPJ:  cnpj,
				CredentialID: credentialID,
			}); err != nil {
				return fmt.Errorf("erro ao atribuir credencial: %w", err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Credencial atribuída com sucesso.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&cnpj, "cnpj", "c", "", "CNPJ da empresa")
	cmd.Flags().StringVar(&credentialID, "credential-id", "", "ID da credencial")
	_ = cmd.MarkFlagRequired("cnpj")
	_ = cmd.MarkFlagRequired("credential-id")
	return cmd
}

// resolveCompanySyncStartFlags collapses the three shortcut flags
// (last-12-months, last-5-years) into the since_date policy.
func resolveCompanySyncStartFlags(policy, date string, last12Months, last5Years bool) (string, string) {
	now := time.Now()
	switch {
	case last12Months:
		return "since_date", now.AddDate(-1, 0, 0).Format("2006-01-02")
	case last5Years:
		return "since_date", now.AddDate(-5, 0, 0).Format("2006-01-02")
	default:
		return policy, date
	}
}
