package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func newNFeBuscarCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var chaveFlag string
	cmd := &cobra.Command{
		Use:   "buscar",
		Short: "Busca uma NF-e na SEFAZ pela chave de acesso",
		Long: "Pede à distribuição da SEFAZ uma NF-e pela chave de acesso e salva o que vier, " +
			"sem mexer no NSU da sincronização. Gasta uma das 20 consultas por hora de NF-e.",
		RunE: func(cmd *cobra.Command, args []string) error {
			chave, err := dfe.ParseAccessKey(chaveFlag)
			if err != nil {
				return fmt.Errorf("chave de acesso inválida: %w", err)
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Buscando NF-e %s...\n", formatChaveDFe(chave))
			res, err := application.NFe.FetchByChave(cmd.Context(), app.FetchByChaveInput{
				CNPJ:        *cnpjFlag,
				ChaveAcesso: chave.String(),
			})
			if err != nil {
				return pullError(out, err)
			}

			switch res.Outcome {
			case app.FetchOutcomeCompleta:
				_, _ = fmt.Fprintln(out, "NF-e completa salva.")
			case app.FetchOutcomeResumo:
				_, _ = fmt.Fprintf(out, "Resumo salvo. %s\n", res.Message)
			default:
				_, _ = fmt.Fprintf(out, "NF-e não localizada. %s\n", res.Message)
			}
			_, _ = fmt.Fprintf(out, "Consultas na última hora: %d/%d\n", res.RequestsLastHour, res.RequestBudget)
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso da NF-e")
	_ = cmd.MarkFlagRequired("chave")
	return cmd
}

// formatChaveDFe prints a 44-digit access key as the DANFE does: 11 groups
// of 4 digits separated by spaces.
func formatChaveDFe(chave dfe.AccessKey) string {
	s := chave.String()
	groups := make([]string, 0, len(s)/4)
	for i := 0; i+4 <= len(s); i += 4 {
		groups = append(groups, s[i:i+4])
	}
	return strings.Join(groups, " ")
}
