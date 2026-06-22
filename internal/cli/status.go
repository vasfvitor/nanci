package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

var statusCNPJ string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Mostra um resumo da situação da empresa",
	RunE: func(cmd *cobra.Command, args []string) error {
		application, err := newApp()
		if err != nil {
			return fmt.Errorf("inicializar: %w", err)
		}
		defer application.Close()

		result, err := application.Status(cmd.Context(), statusCNPJ)
		if err != nil {
			return fmt.Errorf("erro: %w", err)
		}

		fmt.Printf("Status para: %s (%s)\n", result.CompanyName, cnpj.Format(result.CNPJ))
		fmt.Printf("Ambiente: %s\n", result.Environment)
		fmt.Printf("CNPJ consultado: %s\n", cnpj.Format(result.ConsultationCNPJ))
		fmt.Printf("Último NSU consultado: %d\n", result.LastProcessedNSU)
		if result.LastFoundNSU != nil {
			fmt.Printf("Último NSU com documento: %d\n", *result.LastFoundNSU)
		} else {
			fmt.Printf("Último NSU com documento: -\n")
		}
		if result.LastSyncAt != nil {
			fmt.Printf("Última sincronização: %s\n", result.LastSyncAt.Format("2006-01-02 15:04:05"))
		}
		if result.LastRunStatus != "" {
			fmt.Printf("Última execução: %s", result.LastRunStatus)
			if result.LastRunStopReason != "" {
				fmt.Printf(" (%s)", result.LastRunStopReason)
			}
			fmt.Println()
		}
		fmt.Println("\nEstatísticas de Documentos:")
		fmt.Printf("  Notas Emitidas (Prestadas): %d\n", result.TotalEmitidas)
		fmt.Printf("  Notas Tomadas: %d\n", result.TotalTomadas)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&statusCNPJ, "cnpj", "c", "", "CNPJ da empresa")
	_ = statusCmd.MarkFlagRequired("cnpj")
}
