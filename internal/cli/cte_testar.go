package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newCTeTestarConexaoCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "testar-conexao",
		Short: "Testa o certificado e a conexão TLS com a SEFAZ (CT-e) sem consumir consultas",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			result, err := application.CTe.TestConnection(cmd.Context(), *cnpjFlag)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			out := cmd.OutOrStdout()
			if result.CertLoaded {
				_, _ = fmt.Fprintf(out, "Certificado: carregado (%s, válido até %s)\n", result.CertSubject, dashIfEmpty(result.CertExpiration))
			} else {
				_, _ = fmt.Fprintln(out, "Certificado: não carregado")
			}
			if result.EndpointReached {
				_, _ = fmt.Fprintln(out, "Conexão TLS com a SEFAZ: ok")
			} else {
				_, _ = fmt.Fprintln(out, "Conexão TLS com a SEFAZ: falhou")
			}
			_, _ = fmt.Fprintln(out, result.StatusExplanation)
			_, _ = fmt.Fprintln(out, "Nenhuma consulta foi consumida.")

			if !result.EndpointReached {
				return errors.New("teste de conexão com a SEFAZ falhou")
			}
			return nil
		},
	}
}
