package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

var (
	companyCNPJ            string
	companyName            string
	companyCert            string
	companyEnv             string
	companyCredentialID    string
	companyCredentialLabel string
	companySyncStartPolicy string
	companySyncStartDate   string
	companyLast12Months    bool
	companyLast5Years      bool
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
		application, cleanup, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer cleanup()

		environment, err := nfse.ParseEnvironment(companyEnv)
		if err != nil {
			return fmt.Errorf("erro no ambiente: %w", err)
		}
		rawSyncStartPolicy, rawSyncStartDate := resolveCompanySyncStartFlags()
		policy, date, err := app.ParseSyncStartPolicyInput(rawSyncStartPolicy, rawSyncStartDate)
		if err != nil {
			return fmt.Errorf("erro na politica de sincronização: %w", err)
		}

		if err := application.AddCompany(cmd.Context(), app.AddCompanyInput{
			CNPJ:            companyCNPJ,
			Name:            companyName,
			CredentialID:    companyCredentialID,
			CredentialLabel: companyCredentialLabel,
			CertPath:        companyCert,
			Environment:     environment,
			SyncStartPolicy: policy,
			SyncStartDate:   date,
		}); err != nil {
			return fmt.Errorf("erro ao adicionar empresa: %w", err)
		}

		cleanedCNPJ := cnpj.Clean(companyCNPJ)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Empresa '%s' (%s) adicionada com sucesso.\n", companyName, cnpj.Format(cleanedCNPJ))
		return nil
	},
}

var companyListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista todas as empresas",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, cleanup, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer cleanup()

		companies, err := application.ListCompanies(cmd.Context())
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
			_, _ = fmt.Fprintf(out, "%-20s %-24s %-18s %-15s\n", cnpj.Format(c.CNPJ), c.Name, c.CredentialLabel, c.Environment)
		}
		return nil
	},
}

var companyAssignCredentialCmd = &cobra.Command{
	Use:   "assign-credential",
	Short: "Atribui uma credencial existente a uma empresa",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, cleanup, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer cleanup()

		if err := application.AssignCredentialToCompany(cmd.Context(), app.AssignCredentialInput{
			CompanyCNPJ:  companyCNPJ,
			CredentialID: assignCredentialID,
		}); err != nil {
			return fmt.Errorf("erro ao atribuir credencial: %w", err)
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Credencial atribuída com sucesso.")
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
	companyAddCmd.Flags().StringVar(&companySyncStartPolicy, "sync-start-policy", "from_now", "Política inicial: all, since_date ou from_now")
	companyAddCmd.Flags().StringVar(&companySyncStartDate, "sync-start-date", "", "Data de corte inicial YYYY-MM-DD para since_date")
	companyAddCmd.Flags().BoolVar(&companyLast12Months, "last-12-months", false, "Atalho para importar somente os últimos 12 meses no primeiro sync")
	companyAddCmd.Flags().BoolVar(&companyLast5Years, "last-5-years", false, "Atalho para importar somente os últimos 5 anos no primeiro sync")

	_ = companyAddCmd.MarkFlagRequired("cnpj")
	_ = companyAddCmd.MarkFlagRequired("name")

	companyAssignCredentialCmd.Flags().StringVarP(&companyCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	companyAssignCredentialCmd.Flags().StringVar(&assignCredentialID, "credential-id", "", "ID da credencial")
	_ = companyAssignCredentialCmd.MarkFlagRequired("cnpj")
	_ = companyAssignCredentialCmd.MarkFlagRequired("credential-id")
}

func resolveCompanySyncStartFlags() (string, string) {
	now := time.Now()
	switch {
	case companyLast12Months:
		return "since_date", now.AddDate(-1, 0, 0).Format("2006-01-02")
	case companyLast5Years:
		return "since_date", now.AddDate(-5, 0, 0).Format("2006-01-02")
	default:
		return companySyncStartPolicy, companySyncStartDate
	}
}
