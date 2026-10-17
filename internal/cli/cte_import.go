package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
)

func newCTeImportCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "import <arquivo>...",
		Short: "Importa arquivos XML de CT-e e de eventos sem consultar a SEFAZ",
		Long: "Importa arquivos XML de CT-e (CT-e, CT-e OS, GTV-e e CT-e Simplificado) e de eventos " +
			"que a empresa já tem. Informe arquivos, não pastas. Um arquivo rejeitado não impede a importação dos outros.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkImportFiles(args); err != nil {
				return err
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			res, err := application.CTe.ImportXML(cmd.Context(), app.ImportXMLInput{CNPJ: *cnpjFlag, Files: args})
			if err != nil {
				return fmt.Errorf("erro ao importar XML: %w", err)
			}
			return printImportResult(cmd.OutOrStdout(), res)
		},
	}
}
