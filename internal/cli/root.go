package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/danfse/godanfsev2"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

var (
	verbose bool
	trace   bool
)

// rootCmd is the base command when nanci is called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "nanci",
	Short: "CLI para sincronização de XMLs de NFS-e Nacional",
	Long: `nanci (nfse-sync) sincroniza documentos fiscais da API ADN (NFS-e Nacional)
usando certificado digital A1. Suporta extração de retenções e relatórios.`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute(ctx context.Context) int {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Habilita log detalhado (debug)")
	rootCmd.PersistentFlags().BoolVar(&trace, "trace", false, "Habilita log de rastreamento extremo (trace)")
}

// newApp is a helper for commands that need the App instance.
// It also injects the terminal-based CredentialProvider.
func newApp() (*app.App, error) {
	if err := app.LoadRuntimeEnv(); err != nil {
		return nil, fmt.Errorf("falha ao carregar runtime: %w", err)
	}

	// If env var is set, it overrides the flag
	if os.Getenv("NANCI_TRACE") == "1" {
		trace = true
	}
	log := logger.New(verbose, trace)

	application, err := app.NewRuntime(app.RuntimeOptions{
		Log: log,
		CredentialProvider: app.KeyringCredentialProvider{
			Fallback: TerminalCredentialProvider{In: os.Stdin, Out: os.Stderr},
		},
		DANFSeRenderer: godanfsev2.New(),
		RunMigrations:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("configurar aplicação: %w", err)
	}
	return application, nil
}
