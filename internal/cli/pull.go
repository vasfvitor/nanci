package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

var pullCNPJ string

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Sincroniza documentos fiscais da API ADN",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		result, err := application.Pull(cmd.Context(), app.PullInput{
			CNPJ: pullCNPJ,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Iniciando sincronização para %s (%s)\n",
			result.CompanyName, cnpj.Format(result.CNPJ))
		fmt.Printf("Sincronização concluída em %v\n", result.Duration.Round(1e6))
		fmt.Printf("Documentos encontrados: %d | Erros: %d\n",
			result.DocumentsFound, result.Errors)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVarP(&pullCNPJ, "cnpj", "c", "", "CNPJ da empresa para sincronizar")
	pullCmd.MarkFlagRequired("cnpj")
}
