package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// newResetCommand restarts the company's NFS-e sync from NSU 0, keeping the
// documents. Without --confirmar it only prints where the sync stands.
func newResetCommand(env CommandEnv) *cobra.Command {
	var (
		cnpjFlag      string
		confirmarFlag bool
	)
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reinicia a sincronização da NFS-e desde o NSU 0 (simulação sem --confirmar)",
		Long: "Reinicia a sincronização da NFS-e da empresa desde o NSU 0, nos dois ambientes, e libera a troca " +
			"da data inicial com `nanci company update`. Ao contrário de `nfe reset` e `cte reset`, nenhum " +
			"documento é removido: o próximo `nanci pull` percorre a distribuição de novo, a partir da data " +
			"inicial da empresa. Um bloqueio do ADN em vigor continua valendo.",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			// The status before the reset names the company and the cursor
			// that goes back to 0, in both paths.
			status, err := application.SyncManager.Status(cmd.Context(), cnpjFlag)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}

			out := cmd.OutOrStdout()
			if !confirmarFlag {
				if env.jsonOutput() {
					return printJSON(out, newNFSeResetOutput(status, false))
				}
				_, _ = fmt.Fprintf(out, "Redefinição da NFS-e para: %s (%s), ambiente %s\n", status.CompanyName, cnpj.Format(status.CNPJ), status.Environment)
				_, _ = fmt.Fprintf(out, "  Último NSU consultado: %d (voltaria ao NSU 0)\n", status.LastProcessedNSU)
				_, _ = fmt.Fprintln(out, "  Os documentos já baixados ficam.")
				_, _ = fmt.Fprintln(out, "\nNada foi alterado. Use --confirmar para redefinir.")
				return nil
			}

			if err := application.SyncManager.ResetSyncState(cmd.Context(), sync.ResetSyncInput{
				CNPJ:   cnpjFlag,
				Source: syncstate.SyncSourceNFSe,
			}); err != nil {
				if errors.Is(err, sync.ErrSyncRunning) {
					return fmt.Errorf("erro: %w; aguarde a sincronização atual terminar", err)
				}
				return fmt.Errorf("erro: %w", err)
			}
			if env.jsonOutput() {
				return printJSON(out, newNFSeResetOutput(status, true))
			}
			_, _ = fmt.Fprintf(out, "NFS-e redefinida para: %s (%s), ambiente %s\n", status.CompanyName, cnpj.Format(status.CNPJ), status.Environment)
			_, _ = fmt.Fprintf(out, "  Último NSU consultado antes da redefinição: %d\n", status.LastProcessedNSU)
			_, _ = fmt.Fprintln(out, "  Os documentos já baixados ficam.")
			_, _ = fmt.Fprintln(out, "\nO próximo `nanci pull` baixa a distribuição desde o NSU 0. A data inicial da empresa já pode ser alterada com `nanci company update`.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa")
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Redefine de fato; sem esta flag nada é alterado")
	_ = cmd.MarkFlagRequired("cnpj")
	return cmd
}
